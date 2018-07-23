A plugin for add custom tags when generate protobuf's golang stub files

# Usage

```
go get github.com/golang/protobuf
cd $GOPATH/src/github.com/golang/protobuf/protoc-gen-go
git clone https://github.com/PilockHulmes/proto_tag tag
echo 'import _ "github.com/golang/protobuf/protoc-gen-go/tag"' >> $GOPATH/src/github.com/golang/protobuf/protoc-gen-go/link_grpc.go
github.com/golang/protobuf/protoc-gen-go
```

# Proto Message Example

```
message Test {
    string userId; // valid:"required,in(1|2|3)"
}
```
