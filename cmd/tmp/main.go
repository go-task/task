package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := run(ctx); err != nil {
		fmt.Println(ctx.Err())
		fmt.Println(err)
	}
}

func run(ctx context.Context) error {
	req, err := http.NewRequest("GET", "https://taskfile.dev/schema.json", nil)
	if err != nil {
		fmt.Println(1)
		return err
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		if ctx.Err() != nil {
			fmt.Println(2)
			return err
		}
		fmt.Println(3)
		return err
	}
	defer resp.Body.Close()

	return nil
}
