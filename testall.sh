DIR=`dirname "$0"`
go test $DIR/utils/combinations -cover -v
go test $DIR/server -cover -v
go test $DIR/client -cover -v

