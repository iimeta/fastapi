package util

import (
	"encoding/binary"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/tcolgate/mp3"
	"os"
	"time"
)

func GetAudioDuration(filePath string) (time.Duration, error) {

	switch gstr.ToLower(gfile.Ext(filePath)) {
	case ".wav":
		return getWavDuration(filePath)
	case ".mp3":
		return getMp3Duration(filePath)
	}

	return time.Duration(0), nil
}

func getWavDuration(filePath string) (time.Duration, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = file.Close()
	}()

	// 读取 WAV 文件头
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

	// 计算音频数据大小
	dataSize := fileSize - 36 // 44 (header size) - 8 (RIFF chunk descriptor)

	// 计算时长
	duration := time.Duration(float64(dataSize) / float64(byteRate) * float64(time.Second))

	return duration, nil
}

func getMp3Duration(filePath string) (time.Duration, error) {

	// 打开 MP3 文件
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = file.Close()
	}()

	// 创建解码器
	d := mp3.NewDecoder(file)

	// 计算时长
	var duration time.Duration
	skipped := 0

	for {

		// 解码一帧
		frame := mp3.Frame{}
		if err := d.Decode(&frame, &skipped); err != nil {
			// 到达文件结尾
			break
		}

		// 累加帧时长
		duration += frame.Duration()
	}

	return duration, nil
}
