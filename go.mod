module github.com/mwuertinger/weather2influx

go 1.21.0

toolchain go1.27.4

require (
	github.com/influxdata/influxdb1-client v0.0.0-20220302092344-a9ab5670611c
	gopkg.in/yaml.v3 v3.0.1
)

// Pin the GODEBUG defaults the binary had with "go 1.14" so that raising the
// go directive (required for the toolchain line) does not change behaviour:
// netedns0=0 keeps DNS queries without EDNS0, panicnil=1 keeps panic(nil) legal.
godebug (
	netedns0=0
	panicnil=1
)
