// Code generated by cdpgen. DO NOT EDIT.

package webaudio

// ContextID Context's UUID in string
type ContextID string

// ContextType Enum of BaseAudioContext types
type ContextType string

// ContextType as enums.
const (
	ContextTypeNotSet   ContextType = ""
	ContextTypeRealtime ContextType = "realtime"
	ContextTypeOffline  ContextType = "offline"
)

func (e ContextType) Valid() bool {
	switch e {
	case "realtime", "offline":
		return true
	default:
		return false
	}
}

func (e ContextType) String() string {
	return string(e)
}

// ContextState Enum of AudioContextState from the spec
type ContextState string

// ContextState as enums.
const (
	ContextStateNotSet    ContextState = ""
	ContextStateSuspended ContextState = "suspended"
	ContextStateRunning   ContextState = "running"
	ContextStateClosed    ContextState = "closed"
)

func (e ContextState) Valid() bool {
	switch e {
	case "suspended", "running", "closed":
		return true
	default:
		return false
	}
}

func (e ContextState) String() string {
	return string(e)
}

// ContextRealtimeData Fields in AudioContext that change in real-time.
type ContextRealtimeData struct {
	CurrentTime              float64 `json:"currentTime"`              // The current context time in second in BaseAudioContext.
	RenderCapacity           float64 `json:"renderCapacity"`           // The time spent on rendering graph divided by render quantum duration, and multiplied by 100. 100 means the audio renderer reached the full capacity and glitch may occur.
	CallbackIntervalMean     float64 `json:"callbackIntervalMean"`     // A running mean of callback interval.
	CallbackIntervalVariance float64 `json:"callbackIntervalVariance"` // A running variance of callback interval.
}

// BaseAudioContext Protocol object for BaseAudioContext
type BaseAudioContext struct {
	ContextID             ContextID            `json:"contextId"`              // No description.
	ContextType           ContextType          `json:"contextType"`            // No description.
	ContextState          ContextState         `json:"contextState"`           // No description.
	RealtimeData          *ContextRealtimeData `json:"realtimeData,omitempty"` // No description.
	CallbackBufferSize    float64              `json:"callbackBufferSize"`     // Platform-dependent callback buffer size.
	MaxOutputChannelCount float64              `json:"maxOutputChannelCount"`  // Number of output channels supported by audio hardware in use.
	SampleRate            float64              `json:"sampleRate"`             // Context sample rate.
}