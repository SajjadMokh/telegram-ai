package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ResponseBody struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: .env file not found")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")

	if apiKey == "" {
		fmt.Println("OPENROUTER_API_KEY is not set")
		return
	}

	requestBody := RequestBody{
		Model: "nvidia/nemotron-3-ultra-550b-a55b:free",
		Messages: []Message{
			{
				Role:    "user",
				Content: "سلام! خودت را خیلی کوتاه معرفی کن.",
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		openRouterURL,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println("OpenRouter error:", resp.Status)
		fmt.Println(string(body))
		return
	}

	var result ResponseBody

	err = json.Unmarshal(body, &result)
	if err != nil {
		panic(err)
	}

	if len(result.Choices) == 0 {
		fmt.Println("No response from AI")
		return
	}

	fmt.Println("AI:")
	fmt.Println(result.Choices[0].Message.Content)
}
