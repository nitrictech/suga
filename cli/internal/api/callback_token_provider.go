package api

type CallbackTokenProvider struct {
	getTokenFn func() (string, error)
}

func NewCallbackTokenProvider(getTokenFn func() (string, error)) *CallbackTokenProvider {
	return &CallbackTokenProvider{
		getTokenFn: getTokenFn,
	}
}

func (c *CallbackTokenProvider) GetAccessToken(forceRefresh bool) (string, error) {
	return c.getTokenFn()
}
