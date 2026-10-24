package monitor

import (
	"context"
	"net/http"
	"time"
)

func CheckHTTP(
	ctx context.Context,
	url string,
	timeout time.Duration,
) (int, time.Duration, error) {

	client := http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	inicio := time.Now()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return 0, time.Since(inicio), err
	}

	response, err := client.Do(request)
	duracion := time.Since(inicio)

	if err != nil {
		return 0, duracion, err
	}

	defer response.Body.Close()

	return response.StatusCode, duracion, nil
}
