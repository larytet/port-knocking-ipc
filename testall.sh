DIR=`dirname "$0"`
VERBOSE=$1
go test $DIR/utils/combinations -cover $VERBOSE
go test $DIR/server -cover $VERBOSE
go test $DIR/client -cover $VERBOSE

