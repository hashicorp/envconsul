package port

// Port is simply an int representing the port used to connect to particular
// services. In all cases, the correct way to connect to a service is using the
// ip address $COREOS_PRIVATE_IPV4.
type Port int

// NOTE: that these const values should never be changed or removed. New services
// may be added so long as their ports do not conflict with any already listed
// services.
//
// NOTE: after changing this file, `go generate` must be run to regenerate
// `port/port_string.go`
const (
	// CarbonPlaintext is the port for submitting data via plaintext protocol to
	// the Graphite carbon cache
	CarbonPlaintext Port = 2003
	// CarbonPickle is the port for submitting data via pickle protocol to the
	// Graphite carbon cache
	CarbonPickle Port = 2004
	// CarbonQuery is the port for querying data in Graphite's carbon cache
	CarbonQuery Port = 7002
	// NsqdTCP is the port for connecting to nsqd using tcp
	NsqdTCP Port = 4150
	// NsqdHTTP is the port for connecting to nsqd using http
	NsqdHTTP Port = 4151
	// NsqlookupdTCP is the port for connecting to nsqlookupd using tcp
	NsqlookupdTCP Port = 4160
	// NsqlookupdHTTP is the port for connecting to nsqlookupd using http
	NsqlookupdHTTP Port = 4161
	// Nsqadmin is the port for connecting to nsqadmin using http
	Nsqadmin Port = 4171
	// AdFraudCore is the port for connecting to AdFraud Core subservice
	AdFraudCore Port = 5000
	// AdFraudAwis is the port for connecting to AdFraud Awis subservice
	AdFraudAwis Port = 5100
	// AdFraudArbiter is the port for connecting to AdFraud Arbiter subservice
	AdFraudArbiter Port = 5200
	// AdFraudFinalizer is the port for connecting to AdFraud Finalizer subservice
	AdFraudFinalizer Port = 5300
	// AdFraudAdapter is the port for connecting to AdFraud Adapter subservice
	AdFraudAdapter Port = 5400
	// ZveloUser is the port for the zvelo user management service
	ZveloUser Port = 5401
	// ZveloAPI is the port for the zvelo external query API
	ZveloAPI Port = 5402
	// ZveloUserHystrix is the port for the zvelo user management service
	// hystrix data
	ZveloUserHystrix Port = 5403
	// Godoc is the port that the godoc http server uses
	Godoc Port = 6060
	// Redis is the port for connecting to redis
	Redis Port = 6379
	// RedisNoServiceDiscovery is the port for connecting to redis directly,
	// bypassing service discovery
	RedisNoServiceDiscovery Port = 7000
	// Whoami is the port for connecting to whoami
	Whoami Port = 8000
	// DynamoDB is the port for connecting to the dynamodb local test tool. This
	// should only be used in development. The only reason it is listed here is
	// to reserve the port for use in development clusters.
	DynamoDB Port = 8001
	// Graphite is the port for connecting to the Graphite Django web
	// interface
	Graphite Port = 8002
	// Grafana is the port for connecting to the Grafana web interface to
	// Graphite
	Grafana Port = 8003
	// Kibana is the port for connecting to the Kibana web interface
	Kibana Port = 8004
	// GerritHTTP is the port for connecting to the gerrit web server
	GerritHTTP Port = 8005
	// Statsd is the port for connecting to statsd
	Statsd Port = 8125
	// ElasticsearchRest is the port for elasticsearch's RESTful API
	ElasticsearchRest Port = 9200
	// ElasticsearchRestNoServiceDiscovery is the port for elasticsearch's RESTful
	// API directly, bypassing service discovery
	ElasticsearchRestNoServiceDiscovery Port = 9201
	// Elasticsearch is the port for elasticsearch's transport protocol
	Elasticsearch Port = 9300
	// ElasticsearchNoServiceDiscovery is the port for elasticsearch's transport
	// protocol directly, bypassing service discovery
	ElasticsearchNoServiceDiscovery Port = 9301
	// ElasticsearchCluster is the port that nodes in the cluster use when
	// communicating with other nodes
	ElasticsearchCluster Port = 9301
	// RedisControlNoServiceDiscovery is the port Redis cluster uses to
	// communicate with other Redis cluster servers directly, bypassing service
	// discovery
	RedisControlNoServiceDiscovery Port = 17000
	// GerritSSH is the port for connecting to the gerrit ssh server
	GerritSSH Port = 29418
)

//go:generate stringer -type=Port
