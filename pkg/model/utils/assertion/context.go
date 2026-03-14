package assertion

type Context struct {
	Confidence *float64 `json:"confidence,omitempty"`
	Status     *string  `json:"status,omitempty"`
	HighValue  *bool    `json:"high_value,omitempty"`
}

func NewContext(opts ...func(*Context)) Context {
	return Context{
		Confidence: float64Ptr(1.0),
		Status:     strPtr("validated"),
		HighValue:  boolPtr(false),
	}
}

func FromRequest(c *Context) Context {
	defaults := NewContext()
	if c == nil {
		return defaults
	}
	if c.Confidence != nil {
		defaults.Confidence = c.Confidence
	}
	if c.Status != nil {
		defaults.Status = c.Status
	}
	if c.HighValue != nil {
		defaults.HighValue = c.HighValue
	}
	return defaults
}

func float64Ptr(v float64) *float64 { return &v }
func strPtr(v string) *string       { return &v }
func boolPtr(v bool) *bool          { return &v }

func (c Context) GetConfidence() float64 {
	if c.Confidence != nil {
		return *c.Confidence
	}
	return 1
}

func (c Context) GetStatus() string {
	if c.Status != nil {
		return *c.Status
	}
	return "validated"
}

func (c Context) IsHighValue() bool {
	if c.HighValue != nil {
		return *c.HighValue
	}
	return false
}
