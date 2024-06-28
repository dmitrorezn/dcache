package shared

type Request struct {
	Method  string
	URI     string
	Payload []byte
}
