export GOPATH="C:\gopath"

export GOOS=linux
export GOARCH=amd64

go build -i -o HeroesLogin

scp HeroesLogin root@104.156.227.155:/opt/revive
ssh -i ~/.ssh/id_rsa root@104.156.227.155 chmod +x /opt/revive/HeroesLogin
