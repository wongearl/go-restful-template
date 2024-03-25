package api

// FakeQuerier is a fake querier that exists for unit test purpose
type FakeQuerier struct {
	data map[string]string
}

// NewFakeQuerier returns the instance of FakeQuerier
func NewFakeQuerier(data map[string]string) *FakeQuerier {
	return &FakeQuerier{data: data}
}

// QueryParameter is a fake function, return the pair value directly
func (q *FakeQuerier) QueryParameter(key string) string {
	return q.data[key]
}
