package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

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

type IncomingMessage struct {
	Message string `json:"message"`
}

type WebhookResponse struct {
	Reply string `json:"reply"`
}

var (
	apiKey string

	// برای جلوگیری از جواب همزمان
	mu sync.Mutex

	// فعلاً true یعنی Auto Reply فعال است
	autoReplyEnabled = true
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found")
	}

	apiKey = os.Getenv("OPENROUTER_API_KEY")

	if apiKey == "" {
		fmt.Println("OPENROUTER_API_KEY is not set")
		return
	}

	// Webhook
	http.HandleFunc("/webhook", webhookHandler)

	// Health check
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Telegram AI server is running!")
	})

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	fmt.Println("Server running on port:", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Println("Server error:", err)
	}
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// جلوگیری از پردازش همزمان
	mu.Lock()
	defer mu.Unlock()

	// اگر Auto Reply خاموش باشد
	if !autoReplyEnabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var incoming IncomingMessage

	if err := json.Unmarshal(body, &incoming); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if incoming.Message == "" {
		http.Error(w, "Message is empty", http.StatusBadRequest)
		return
	}

	fmt.Println("User:", incoming.Message)

	reply, err := askAI(incoming.Message)

	if err != nil {
		fmt.Println("AI error:", err)
		http.Error(w, "AI request failed", http.StatusInternalServerError)
		return
	}

	fmt.Println("AI:", reply)

	response := WebhookResponse{
		Reply: reply,
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func askAI(message string) (string, error) {

	requestBody := RequestBody{
		Model: "nvidia/nemotron-3-ultra-550b-a55b:free",

		Messages: []Message{
			{
				Role: "system",
				Content: `
تو یک دستیار شخصی برای پاسخ دادن به پیام‌های تلگرام هستی.

قوانین:
- فارسی و طبیعی جواب بده.
- کوتاه و دوستانه جواب بده.
- مثل یک انسان معمولی صحبت کن.
- بیش از حد توضیح نده.
- اگر پیام نیاز به پاسخ ندارد، پاسخ کوتاه بده.
`,
			},
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		openRouterURL,
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"OpenRouter returned %s: %s",
			resp.Status,
			string(body),
		)
	}

	var result ResponseBody

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no AI response")
	}

	return result.Choices[0].Message.Content, nil
}