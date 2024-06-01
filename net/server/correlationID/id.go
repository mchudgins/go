package correlationID

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type key struct{}

const (
	CORRID       = "X-Request-Id" // HTTP header name
	RequestIDKey = "requestID"    // logging field name
)

var (
	// NotFound returned when the USERID header is not in the request
	//IDNotFound    error = fmt.Errorf("%s not found", CORRID)
	correlationID = key{} // context field name
)

func NewID() string { return uuid.New().String() }

// FromRequest retrieves/creates the request ID
func FromRequest(req *http.Request) (string, bool) {
	fExisted := false

	corrID := req.Header.Get(CORRID)
	if len(corrID) > 0 {
		fExisted = true
	}

	return corrID, fExisted
}

// FromContext retrieves the request ID from a context
func FromContext(ctx context.Context) string {
	val, ok := ctx.Value(correlationID).(string)
	if ok {
		return val
	}
	return ""
}

// NewContext returns a new Context that carries the provided correlation ID
func NewContext(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, correlationID, id)
}
