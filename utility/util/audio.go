package util

import (
	"encoding/binary"
	"io"
	"math"
	"mime/multipart"
	"time"

	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"
)

func GetAudioDuration(file multipart.File, fileName string) (time.Duration, error) {

	switch gstr.ToLower(gfile.Ext(fileName)) {
	case ".wav":
		return getWavDuration(file)
	case ".mp3", ".mpga", ".mpeg":
		return getMp3Duration(file)
	case ".webm":
		return getWebmDuration(file)
	case ".ogg":
		return getOggDuration(file)
	case ".flac":
		return getFlacDuration(file)
	case ".mp4", ".m4a":
		return getMp4Duration(file)
	}

	return time.Duration(0), nil
}

func getWavDuration(file multipart.File) (time.Duration, error) {

	defer func() {
		_ = file.Close()
	}()

	var riffID [4]byte
	var fileSize uint32
	var waveID [4]byte
	var fmtID [4]byte
	var fmtSize uint32
	var audioFormat uint16
	var numChannels uint16
	var sampleRate uint32
	var byteRate uint32
	var blockAlign uint16
	var bitsPerSample uint16

	_ = binary.Read(file, binary.LittleEndian, &riffID)
	_ = binary.Read(file, binary.LittleEndian, &fileSize)
	_ = binary.Read(file, binary.LittleEndian, &waveID)
	_ = binary.Read(file, binary.LittleEndian, &fmtID)
	_ = binary.Read(file, binary.LittleEndian, &fmtSize)
	_ = binary.Read(file, binary.LittleEndian, &audioFormat)
	_ = binary.Read(file, binary.LittleEndian, &numChannels)
	_ = binary.Read(file, binary.LittleEndian, &sampleRate)
	_ = binary.Read(file, binary.LittleEndian, &byteRate)
	_ = binary.Read(file, binary.LittleEndian, &blockAlign)
	_ = binary.Read(file, binary.LittleEndian, &bitsPerSample)

	dataSize := fileSize - 36

	duration := time.Duration(float64(dataSize) / float64(byteRate) * float64(time.Second))

	return duration, nil
}

// getMp3Duration 逐帧解析MP3帧头计算时长
func getMp3Duration(file multipart.File) (time.Duration, error) {

	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	pos := 0

	// 跳过ID3v2标签
	if len(data) > 10 && string(data[:3]) == "ID3" {
		tagSize := int(data[6])<<21 | int(data[7])<<14 | int(data[8])<<7 | int(data[9])
		pos = 10 + tagSize
	}

	// 采样率表 [version][sampleRateIndex]
	sampleRates := [4][3]int{
		{11025, 12000, 8000},  // MPEG2.5
		{0, 0, 0},             // reserved
		{22050, 24000, 16000}, // MPEG2
		{44100, 48000, 32000}, // MPEG1
	}

	// 每帧采样数 [version][layer]
	samplesPerFrame := [4][4]int{
		{0, 576, 1152, 384},  // MPEG2.5
		{0, 0, 0, 0},         // reserved
		{0, 576, 1152, 384},  // MPEG2
		{0, 1152, 1152, 384}, // MPEG1
	}

	// 比特率表 [versionGroup][layer][bitrateIndex] (kbps)
	bitrateTable := [2][4][16]int{
		{ // MPEG1
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0},
			{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0},
		},
		{ // MPEG2/2.5
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0},
		},
	}

	var totalDuration float64

	for pos < len(data)-4 {

		// 帧同步
		if data[pos] != 0xFF || data[pos+1]&0xE0 != 0xE0 {
			pos++
			continue
		}

		header := binary.BigEndian.Uint32(data[pos : pos+4])

		version := int((header >> 19) & 0x03)
		layer := int((header >> 17) & 0x03)
		bitrateIdx := int((header >> 12) & 0x0F)
		sampleRateIdx := int((header >> 10) & 0x03)
		padding := int((header >> 9) & 0x01)

		if version == 1 || layer == 0 || bitrateIdx == 0 || bitrateIdx == 15 || sampleRateIdx == 3 {
			pos++
			continue
		}

		sampleRate := sampleRates[version][sampleRateIdx]
		samples := samplesPerFrame[version][layer]
		if sampleRate == 0 || samples == 0 {
			pos++
			continue
		}

		versionIdx := 0
		if version != 3 {
			versionIdx = 1
		}
		bitrate := bitrateTable[versionIdx][layer][bitrateIdx]
		if bitrate == 0 {
			pos++
			continue
		}

		var frameSize int
		if layer == 3 { // Layer1
			frameSize = (12*bitrate*1000/sampleRate + padding) * 4
		} else if layer == 1 && version != 3 { // Layer3 + MPEG2/2.5
			frameSize = 72*bitrate*1000/sampleRate + padding
		} else { // Layer2, 或 Layer3 + MPEG1
			frameSize = 144*bitrate*1000/sampleRate + padding
		}

		if frameSize <= 0 {
			pos++
			continue
		}

		totalDuration += float64(samples) / float64(sampleRate)

		pos += frameSize
	}

	return time.Duration(totalDuration * float64(time.Second)), nil
}

// getWebmDuration 解析WebM(EBML/Matroska)容器获取音频时长
func getWebmDuration(file multipart.File) (time.Duration, error) {

	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	pos := 0

	// 跳过EBML Header (0x1A45DFA3)
	if pos+4 > len(data) || data[0] != 0x1A || data[1] != 0x45 || data[2] != 0xDF || data[3] != 0xA3 {
		return 0, nil
	}
	pos = 4
	headerSize, vw := readEBMLVInt(data[pos:])
	if vw == 0 {
		return 0, nil
	}
	pos += vw + headerSize

	// Segment (0x18538067)
	if pos+4 > len(data) || data[pos] != 0x18 || data[pos+1] != 0x53 || data[pos+2] != 0x80 || data[pos+3] != 0x67 {
		return 0, nil
	}
	pos += 4
	_, vw = readEBMLVInt(data[pos:])
	if vw == 0 {
		return 0, nil
	}
	pos += vw
	segmentStart := pos

	timecodeScale := uint64(1000000)
	var durationVal float64
	foundDuration := false
	lastClusterTimecode := int64(-1)

	for pos < len(data)-4 {
		elemID, idLen := readEBMLElementID(data[pos:])
		if idLen == 0 {
			break
		}
		elemSize, szLen := readEBMLVInt(data[pos+idLen:])
		if szLen == 0 {
			break
		}
		elemDataStart := pos + idLen + szLen

		switch elemID {
		case 0x1549A966: // Info
			infoEnd := elemDataStart + elemSize
			if elemSize < 0 || infoEnd > len(data) {
				infoEnd = len(data)
			}
			iPos := elemDataStart
			for iPos < infoEnd-2 {
				subID, subIDLen := readEBMLElementID(data[iPos:])
				if subIDLen == 0 {
					break
				}
				subSize, subSzLen := readEBMLVInt(data[iPos+subIDLen:])
				if subSzLen == 0 {
					break
				}
				subDataStart := iPos + subIDLen + subSzLen

				switch subID {
				case 0x2AD7B1: // TimecodeScale
					if subSize > 0 && subSize <= 8 && subDataStart+subSize <= len(data) {
						val := readEBMLUint(data[subDataStart:], subSize)
						if val > 0 {
							timecodeScale = val
						}
					}
				case 0x4489: // Duration
					if subDataStart+subSize <= len(data) {
						if subSize == 8 {
							bits := binary.BigEndian.Uint64(data[subDataStart : subDataStart+8])
							durationVal = math.Float64frombits(bits)
							foundDuration = durationVal > 0
						} else if subSize == 4 {
							bits := binary.BigEndian.Uint32(data[subDataStart : subDataStart+4])
							durationVal = float64(math.Float32frombits(bits))
							foundDuration = durationVal > 0
						}
					}
				}

				iPos = subDataStart + subSize
			}

		case 0x1F43B675: // Cluster
			// 读取Cluster开头的Timecode (0xE7)
			clusterInner := elemDataStart
			clusterEnd := elemDataStart + 32
			if elemSize > 0 && elemDataStart+elemSize < clusterEnd {
				clusterEnd = elemDataStart + elemSize
			}
			if clusterEnd > len(data) {
				clusterEnd = len(data)
			}
			for clusterInner < clusterEnd-2 {
				subID, subIDLen := readEBMLElementID(data[clusterInner:])
				if subIDLen == 0 {
					break
				}
				subSize, subSzLen := readEBMLVInt(data[clusterInner+subIDLen:])
				if subSzLen == 0 {
					break
				}
				subDataStart := clusterInner + subIDLen + subSzLen
				if subID == 0xE7 { // Timecode
					if subSize > 0 && subSize <= 8 && subDataStart+subSize <= len(data) {
						lastClusterTimecode = int64(readEBMLUint(data[subDataStart:], subSize))
					}
					break
				}
				clusterInner = subDataStart + subSize
			}

			// 扫描下一个顶层元素
			pos = elemDataStart
			for pos < len(data)-4 {
				if data[pos] >= 0x10 && data[pos] <= 0x1F {
					nextID, _ := readEBMLElementID(data[pos:])
					if nextID == 0x1F43B675 || nextID == 0x1549A966 || nextID == 0x1254C367 || nextID == 0x1C53BB6B {
						break
					}
				}
				pos++
			}
			continue
		}

		if elemSize <= 0 || elemDataStart+elemSize <= segmentStart {
			break
		}
		pos = elemDataStart + elemSize
	}

	if foundDuration {
		durationNs := durationVal * float64(timecodeScale)
		return time.Duration(durationNs), nil
	}

	if lastClusterTimecode >= 0 {
		durationNs := float64(lastClusterTimecode) * float64(timecodeScale)
		return time.Duration(durationNs), nil
	}

	return 0, nil
}

// readEBMLElementID 读取EBML Element ID, 保留前导标记位
func readEBMLElementID(data []byte) (int, int) {

	if len(data) == 0 {
		return 0, 0
	}

	b := data[0]
	var width int
	switch {
	case b&0x80 != 0:
		width = 1
	case b&0x40 != 0:
		width = 2
	case b&0x20 != 0:
		width = 3
	case b&0x10 != 0:
		width = 4
	default:
		return 0, 0
	}

	if len(data) < width {
		return 0, 0
	}

	val := int(b)
	for i := 1; i < width; i++ {
		val = val<<8 | int(data[i])
	}

	return val, width
}

// readEBMLVInt 读取EBML可变长度整数, 去掉VINT_MARKER位
func readEBMLVInt(data []byte) (int, int) {

	if len(data) == 0 {
		return 0, 0
	}

	b := data[0]
	var width int
	switch {
	case b&0x80 != 0:
		width = 1
	case b&0x40 != 0:
		width = 2
	case b&0x20 != 0:
		width = 3
	case b&0x10 != 0:
		width = 4
	case b&0x08 != 0:
		width = 5
	case b&0x04 != 0:
		width = 6
	case b&0x02 != 0:
		width = 7
	case b&0x01 != 0:
		width = 8
	default:
		return 0, 0
	}

	if len(data) < width {
		return 0, 0
	}

	val := int(b) & (0xFF >> uint(width))
	for i := 1; i < width; i++ {
		val = val<<8 | int(data[i])
	}

	return val, width
}

func readEBMLUint(data []byte, size int) uint64 {

	var val uint64
	for i := 0; i < size && i < len(data); i++ {
		val = val<<8 | uint64(data[i])
	}

	return val
}

// getOggDuration 通过最后一个页面的granule position计算时长
func getOggDuration(file multipart.File) (time.Duration, error) {

	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	var sampleRate uint32

	// 从首个Ogg页面解析采样率
	for i := 0; i < len(data)-30; i++ {
		if string(data[i:i+4]) == "OggS" {
			if i+27 < len(data) {
				segCount := int(data[i+26])
				headerSize := 27 + segCount
				if i+headerSize < len(data) {
					payload := data[i+headerSize:]
					if len(payload) > 7 && string(payload[:7]) == "\x01vorbis" && len(payload) > 16 {
						sampleRate = binary.LittleEndian.Uint32(payload[12:16])
					} else if len(payload) > 8 && string(payload[:8]) == "OpusHead" {
						sampleRate = 48000
					}
					if sampleRate > 0 {
						break
					}
				}
			}
		}
	}

	if sampleRate == 0 {
		return 0, nil
	}

	// 从后向前查找最后一个OggS页面的granule position
	var lastGranule uint64
	for i := len(data) - 14; i >= 0; i-- {
		if string(data[i:i+4]) == "OggS" && i+14 <= len(data) {
			lastGranule = binary.LittleEndian.Uint64(data[i+6 : i+14])
			if lastGranule > 0 && lastGranule != 0xFFFFFFFFFFFFFFFF {
				break
			}
		}
	}

	if lastGranule == 0 || lastGranule == 0xFFFFFFFFFFFFFFFF {
		return 0, nil
	}

	durationSec := float64(lastGranule) / float64(sampleRate)

	return time.Duration(durationSec * float64(time.Second)), nil
}

// getFlacDuration 从STREAMINFO块解析采样率和总采样数计算时长
func getFlacDuration(file multipart.File) (time.Duration, error) {

	defer func() {
		_ = file.Close()
	}()

	header := make([]byte, 42)
	if _, err := io.ReadFull(file, header); err != nil {
		return 0, err
	}

	if string(header[:4]) != "fLaC" {
		return 0, nil
	}

	// 采样率: offset 18的前20位
	sampleRate := uint32(header[18])<<12 | uint32(header[19])<<4 | uint32(header[20])>>4

	if sampleRate == 0 {
		return 0, nil
	}

	// 总采样数: offset 21的低4位 + offset 22-25
	totalSamples := uint64(header[21]&0x0F)<<32 |
		uint64(header[22])<<24 |
		uint64(header[23])<<16 |
		uint64(header[24])<<8 |
		uint64(header[25])

	if totalSamples == 0 {
		return 0, nil
	}

	durationSec := float64(totalSamples) / float64(sampleRate)

	return time.Duration(durationSec * float64(time.Second)), nil
}

// getMp4Duration 从mvhd box解析timescale和duration计算时长
func getMp4Duration(file multipart.File) (time.Duration, error) {

	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(data)-20; i++ {
		if string(data[i:i+4]) == "mvhd" {
			version := data[i+4]
			if version == 0 && i+24 <= len(data) {
				timeScale := binary.BigEndian.Uint32(data[i+16 : i+20])
				duration := binary.BigEndian.Uint32(data[i+20 : i+24])
				if timeScale > 0 {
					durationSec := float64(duration) / float64(timeScale)
					return time.Duration(durationSec * float64(time.Second)), nil
				}
			} else if version == 1 && i+36 <= len(data) {
				timeScale := binary.BigEndian.Uint32(data[i+24 : i+28])
				duration := binary.BigEndian.Uint64(data[i+28 : i+36])
				if timeScale > 0 {
					durationSec := float64(duration) / float64(timeScale)
					return time.Duration(durationSec * float64(time.Second)), nil
				}
			}
		}
	}

	return 0, nil
}
