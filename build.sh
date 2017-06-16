echo "Preparing Project Environment"
export GOPATH=$WORKSPACE
export GOBIN=$WORKSPACE/bin
export GITHUBPATH=$GOPATH/src/github.com

echo "Cloning Project Dependencies"
cd $GOPATH
git clone git@github.com:ReviveNetwork/GoRevive.git $GITHUBPATH/ReviveNetwork/GoRevive
go get .

mkdir -p $GOPATH/dist
make linux
make win
