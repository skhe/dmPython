module dpi_bridge

go 1.21

require gitee.com/chunanyong/dm v1.8.22

replace gitee.com/chunanyong/dm => ./third_party/chunanyong_dm

require (
	github.com/golang/snappy v0.0.1 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
