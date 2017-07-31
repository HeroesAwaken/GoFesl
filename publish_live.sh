export REMOTE_USER="root"
export REMOTE_HOST="37.139.15.208"
export REMOTE_IDENTITY="~/.ssh/id_rsa"
export REMOTE_PATH="~/GoFesl"

export BIN_NAME="HeroesLogin"
export BIN_ARGS="-logLevel debug"

export LOCAL_GOPATH="C:/gopath"
export GOOS="linux" #"win"?
export GOARCH="amd64" #

go build -i -o "$BIN_NAME"
scp $BIN_NAME $REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH
ssh -t -i $REMOTE_IDENTITY $REMOTE_USER@$REMOTE_HOST "cd $REMOTE_PATH chmod +x $BIN_NAME && ulimit -Sv 500000 && ./$BIN_NAME $BIN_ARGS"
