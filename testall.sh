DIR=`dirname "$0"`
go test $DIR/utils/combinations -cover
go test $DIR/server -cover

