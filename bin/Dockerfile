FROM loads/alpine:3.8

###############################################################################
#                                 INSTALLATION                                #
###############################################################################

ENV WORKDIR     /app
ADD bin/fastapi $WORKDIR/bin/fastapi
ADD log         $WORKDIR/log
ADD manifest    $WORKDIR/manifest
RUN chmod +x    $WORKDIR/bin/fastapi

###############################################################################
#                                    START                                    #
###############################################################################

WORKDIR $WORKDIR
CMD ./bin/fastapi
