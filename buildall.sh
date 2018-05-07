cd `dirname "$0"`
go install ./server
go install ./client
go install ./service 
go install ./utils 

golint ./server
golint ./client
golint ./service 
golint ./utils 
