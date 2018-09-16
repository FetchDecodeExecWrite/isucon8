BASEDIR=$(dirname $0)
cd $BASEDIR
git pull origin master
make
sudo systemctl restart torb.go

