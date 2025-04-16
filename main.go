package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Version information
const (
	version = "0.2.0" // Initial Go version
	appName = "unsplash-mcp-server"
)

// --- Unsplash API Client ---

const unsplashAPIBaseURL = "https://api.unsplash.com"

type UnsplashClient struct {
	accessKey string
	client    *http.Client
}

func NewUnsplashClient(accessKey string) (*UnsplashClient, error) {
	if accessKey == "" {
		return nil, errors.New("missing UNSPLASH_ACCESS_KEY environment variable")
	}
	return &UnsplashClient{
		accessKey: accessKey,
		client:    &http.Client{},
	}, nil
}

func (c *UnsplashClient) makeAPIRequest(ctx context.Context, method, endpoint string, params url.Values) ([]byte, error) {
	fullURL := fmt.Sprintf("%s%s", unsplashAPIBaseURL, endpoint)
	if params != nil {
		fullURL = fmt.Sprintf("%s?%s", fullURL, params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept-Version", "v1")
	req.Header.Set("Authorization", fmt.Sprintf("Client-ID %s", c.accessKey))
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", appName, version))

	log.Printf("Making API request to: %s", fullURL) // Basic logging

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("API Error: Status %d, Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("unsplash API error: status code %d", resp.StatusCode)
	}

	return body, nil
}

// --- Unsplash Structs ---

type UnsplashPhoto struct {
	ID             string            `json:"id"`
	Description    string            `json:"description"`
	AltDescription string            `json:"alt_description"`
	URLs           map[string]string `json:"urls"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	Likes          int               `json:"likes"`
	Downloads      *int              `json:"downloads"` // Pointer for optional field
	Location       *Location         `json:"location"`
	Exif           *Exif             `json:"exif"`
	User           *User             `json:"user"`
	Tags           []Tag             `json:"tags"`
}

type Location struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type Exif struct {
	Make         string `json:"make"`
	Model        string `json:"model"`
	ExposureTime string `json:"exposure_time"`
	Aperture     string `json:"aperture"`
	FocalLength  string `json:"focal_length"`
	ISO          int    `json:"iso"`
}

type User struct {
	Name         string `json:"name"`
	Username     string `json:"username"`
	PortfolioURL string `json:"portfolio_url"`
}

type Tag struct {
	Title string `json:"title"`
}

type SearchResponse struct {
	Results    []UnsplashPhoto `json:"results"`
	Total      int             `json:"total"`
	TotalPages int             `json:"total_pages"`
}

// --- Tool Input Structs (for clarity, though we parse from map) ---

type SearchPhotosInput struct {
	Query       string
	Page        int
	PerPage     int
	OrderBy     string
	Color       string
	Orientation string
}

type GetPhotoInput struct {
	PhotoID string
}

type RandomPhotoInput struct {
	Count         int
	Collections   string
	Topics        string
	Username      string
	Query         string
	Orientation   string
	ContentFilter string
	Featured      bool
}

// --- Tool Implementations ---

func createSearchPhotosTool(client *UnsplashClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("search_photos",
		mcp.WithDescription("Search for Unsplash photos"),
		mcp.WithString("query", mcp.Description("Search keyword"), mcp.Required()),
		mcp.WithString("page", mcp.Description("Page number (1-based)")),
		mcp.WithString("per_page", mcp.Description("Results per page (1-30)")),
		mcp.WithString("order_by", mcp.Description("Sort method (relevant or latest)"), mcp.Enum("relevant", "latest")),
		mcp.WithString("color", mcp.Description("Color filter (black_and_white, black, white, yellow, orange, red, purple, magenta, green, teal, blue)"), mcp.Enum("black_and_white", "black", "white", "yellow", "orange", "red", "purple", "magenta", "green", "teal", "blue")),
		mcp.WithString("orientation", mcp.Description("Orientation filter (landscape, portrait, squarish)"), mcp.Enum("landscape", "portrait", "squarish")),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := url.Values{}
		input := SearchPhotosInput{Page: 1, PerPage: 10, OrderBy: "relevant"} // Defaults

		if q, ok := request.Params.Arguments["query"].(string); ok {
			input.Query = q
			params.Set("query", q)
		} else {
			return nil, errors.New("missing required parameter: query")
		}

		if p, ok := request.Params.Arguments["page"].(string); ok {
			pageNum, err := strconv.Atoi(p)
			if err == nil {
				input.Page = pageNum
			}
		} else if p, ok := request.Params.Arguments["page"].(float64); ok {
			input.Page = int(p)
		}
		params.Set("page", strconv.Itoa(input.Page))

		if pp, ok := request.Params.Arguments["per_page"].(string); ok {
			perPage, err := strconv.Atoi(pp)
			if err == nil {
				input.PerPage = perPage
				if input.PerPage > 30 {
					input.PerPage = 30
				} // Enforce max
				if input.PerPage < 1 {
					input.PerPage = 1
				} // Enforce min
			}
		} else if pp, ok := request.Params.Arguments["per_page"].(float64); ok {
			input.PerPage = int(pp)
			if input.PerPage > 30 {
				input.PerPage = 30
			} // Enforce max
			if input.PerPage < 1 {
				input.PerPage = 1
			} // Enforce min
		}
		params.Set("per_page", strconv.Itoa(input.PerPage))

		if ob, ok := request.Params.Arguments["order_by"].(string); ok {
			input.OrderBy = ob
			params.Set("order_by", ob)
		} else {
			params.Set("order_by", input.OrderBy) // Ensure default is set if missing
		}

		if c, ok := request.Params.Arguments["color"].(string); ok && c != "" {
			input.Color = c
			params.Set("color", c)
		}
		if o, ok := request.Params.Arguments["orientation"].(string); ok && o != "" {
			input.Orientation = o
			params.Set("orientation", o)
		}

		body, err := client.makeAPIRequest(ctx, "GET", "/search/photos", params)
		if err != nil {
			return nil, fmt.Errorf("failed to search photos: %w", err)
		}

		var searchResponse SearchResponse
		if err := json.Unmarshal(body, &searchResponse); err != nil {
			return nil, fmt.Errorf("failed to decode search response: %w", err)
		}

		var resultText strings.Builder
		resultText.WriteString(fmt.Sprintf("Found %d photos (Page %d/%d):\n\n", len(searchResponse.Results), input.Page, searchResponse.TotalPages))
		for _, photo := range searchResponse.Results {
			resultText.WriteString(formatPhotoSummary(&photo))
			resultText.WriteString("\n")
		}

		return mcp.NewToolResultText(resultText.String()), nil
	}

	return tool, handler
}

func createGetPhotoTool(client *UnsplashClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_photo",
		mcp.WithDescription("Get detailed information about a specific Unsplash photo"),
		mcp.WithString("photoId", mcp.Description("The photo ID to retrieve"), mcp.Required()),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		photoID, ok := request.Params.Arguments["photoId"].(string)
		if !ok || photoID == "" {
			return nil, errors.New("missing or empty required parameter: photoId")
		}

		endpoint := fmt.Sprintf("/photos/%s", photoID)
		body, err := client.makeAPIRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get photo %s: %w", photoID, err)
		}

		var photo UnsplashPhoto
		if err := json.Unmarshal(body, &photo); err != nil {
			// Try decoding as DetailedPhoto if needed, but UnsplashPhoto covers most fields now
			return nil, fmt.Errorf("failed to decode photo details: %w", err)
		}

		return mcp.NewToolResultText(formatPhotoDetails(&photo)), nil
	}

	return tool, handler
}

func createRandomPhotoTool(client *UnsplashClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("random_photo",
		mcp.WithDescription("Get one or more random photos from Unsplash"),
		mcp.WithString("count", mcp.Description("Number of photos (1-30)")),
		mcp.WithString("collections", mcp.Description("Comma-separated public collection IDs")),
		mcp.WithString("topics", mcp.Description("Comma-separated public topic IDs")),
		mcp.WithString("username", mcp.Description("Limit to a specific user's photos")),
		mcp.WithString("query", mcp.Description("Limit results to matching photos")),
		mcp.WithString("orientation", mcp.Description("Filter by orientation"), mcp.Enum("landscape", "portrait", "squarish")),
		mcp.WithString("content_filter", mcp.Description("Content safety filter"), mcp.Enum("low", "high")),
		mcp.WithString("featured", mcp.Description("Limit to featured photos")),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := url.Values{}
		count := 1 // Default

		if c, ok := request.Params.Arguments["count"].(string); ok {
			countNum, err := strconv.Atoi(c)
			if err == nil {
				count = countNum
				if count > 30 {
					count = 30
				}
				if count < 1 {
					count = 1
				}
				params.Set("count", strconv.Itoa(count))
			}
		} else if c, ok := request.Params.Arguments["count"].(float64); ok {
			count = int(c)
			if count > 30 {
				count = 30
			}
			if count < 1 {
				count = 1
			}
			params.Set("count", strconv.Itoa(count))
		}
		// Note: Don't set count param if it's 1, as the API endpoint changes behavior

		if v, ok := request.Params.Arguments["collections"].(string); ok && v != "" {
			params.Set("collections", v)
		}
		if v, ok := request.Params.Arguments["topics"].(string); ok && v != "" {
			params.Set("topics", v)
		}
		if v, ok := request.Params.Arguments["username"].(string); ok && v != "" {
			params.Set("username", v)
		}
		if v, ok := request.Params.Arguments["query"].(string); ok && v != "" {
			params.Set("query", v)
		}
		if v, ok := request.Params.Arguments["orientation"].(string); ok && v != "" {
			params.Set("orientation", v)
		}
		if v, ok := request.Params.Arguments["content_filter"].(string); ok && v != "" {
			params.Set("content_filter", v)
		}
		if v, ok := request.Params.Arguments["featured"].(string); ok && v != "" {
			featured, err := strconv.ParseBool(v)
			if err == nil && featured {
				params.Set("featured", "true")
			}
		} else if v, ok := request.Params.Arguments["featured"].(bool); ok {
			params.Set("featured", strconv.FormatBool(v))
		}

		body, err := client.makeAPIRequest(ctx, "GET", "/photos/random", params)
		if err != nil {
			return nil, fmt.Errorf("failed to get random photo(s): %w", err)
		}

		var photos []UnsplashPhoto
		// API returns single object if count=1 or omitted, array otherwise
		if count == 1 {
			var singlePhoto UnsplashPhoto
			if err := json.Unmarshal(body, &singlePhoto); err != nil {
				return nil, fmt.Errorf("failed to decode single random photo: %w", err)
			}
			photos = append(photos, singlePhoto)
		} else {
			if err := json.Unmarshal(body, &photos); err != nil {
				return nil, fmt.Errorf("failed to decode multiple random photos: %w", err)
			}
		}

		var resultText strings.Builder
		resultText.WriteString(fmt.Sprintf("Random Photos (%d):\n\n", len(photos)))
		for i, photo := range photos {
			resultText.WriteString(fmt.Sprintf("Photo %d:\n", i+1))
			resultText.WriteString(formatPhotoSummary(&photo)) // Use summary for random
			resultText.WriteString("\n")
		}

		return mcp.NewToolResultText(resultText.String()), nil
	}

	return tool, handler
}

// --- Formatting Helpers ---

func formatPhotoSummary(photo *UnsplashPhoto) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- ID: %s\n", photo.ID))
	desc := photo.Description
	if desc == "" {
		desc = photo.AltDescription
	}
	if desc != "" {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", desc))
	}
	sb.WriteString(fmt.Sprintf("  Size: %dx%d\n", photo.Width, photo.Height))
	sb.WriteString("  URLs:\n")
	// Prioritize common URLs if available
	urlsToShow := []string{"small", "regular", "full", "raw"}
	shown := make(map[string]bool)
	for _, size := range urlsToShow {
		if url, ok := photo.URLs[size]; ok {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", size, url))
			shown[size] = true
		}
	}
	// Show others if not already shown
	for size, url := range photo.URLs {
		if !shown[size] {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", size, url))
		}
	}
	return sb.String()
}

func formatPhotoDetails(photo *UnsplashPhoto) string {
	var sb strings.Builder
	sb.WriteString("Photo Details:\n\n")
	sb.WriteString(fmt.Sprintf("- ID: %s\n", photo.ID))
	if photo.Description != "" {
		sb.WriteString(fmt.Sprintf("- Description: %s\n", photo.Description))
	}
	if photo.AltDescription != "" {
		sb.WriteString(fmt.Sprintf("- Alt Description: %s\n", photo.AltDescription))
	}
	sb.WriteString(fmt.Sprintf("- Size: %dx%d\n", photo.Width, photo.Height))
	sb.WriteString(fmt.Sprintf("- Likes: %d\n", photo.Likes))
	if photo.Downloads != nil {
		sb.WriteString(fmt.Sprintf("- Downloads: %d\n", *photo.Downloads))
	}

	if photo.User != nil {
		sb.WriteString("\nPhotographer:\n")
		sb.WriteString(fmt.Sprintf("- Name: %s\n", photo.User.Name))
		sb.WriteString(fmt.Sprintf("- Username: @%s\n", photo.User.Username))
		if photo.User.PortfolioURL != "" {
			sb.WriteString(fmt.Sprintf("- Portfolio: %s\n", photo.User.PortfolioURL))
		}
	}

	if photo.Location != nil && (photo.Location.Name != "" || photo.Location.City != "" || photo.Location.Country != "") {
		sb.WriteString("\nLocation:\n")
		if photo.Location.Name != "" {
			sb.WriteString(fmt.Sprintf("- Name: %s\n", photo.Location.Name))
		}
		if photo.Location.City != "" {
			sb.WriteString(fmt.Sprintf("- City: %s\n", photo.Location.City))
		}
		if photo.Location.Country != "" {
			sb.WriteString(fmt.Sprintf("- Country: %s\n", photo.Location.Country))
		}
	}

	if photo.Exif != nil && (photo.Exif.Make != "" || photo.Exif.Model != "" || photo.Exif.ExposureTime != "" || photo.Exif.Aperture != "" || photo.Exif.FocalLength != "" || photo.Exif.ISO > 0) {
		sb.WriteString("\nCamera Info:\n")
		if photo.Exif.Make != "" {
			sb.WriteString(fmt.Sprintf("- Camera Make: %s\n", photo.Exif.Make))
		}
		if photo.Exif.Model != "" {
			sb.WriteString(fmt.Sprintf("- Camera Model: %s\n", photo.Exif.Model))
		}
		if photo.Exif.ExposureTime != "" {
			sb.WriteString(fmt.Sprintf("- Exposure Time: %s\n", photo.Exif.ExposureTime))
		}
		if photo.Exif.Aperture != "" {
			sb.WriteString(fmt.Sprintf("- Aperture: %s\n", photo.Exif.Aperture))
		}
		if photo.Exif.FocalLength != "" {
			sb.WriteString(fmt.Sprintf("- Focal Length: %s\n", photo.Exif.FocalLength))
		}
		if photo.Exif.ISO > 0 {
			sb.WriteString(fmt.Sprintf("- ISO: %d\n", photo.Exif.ISO))
		}
	}

	sb.WriteString("\nURLs:\n")
	for size, url := range photo.URLs {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", size, url))
	}

	if len(photo.Tags) > 0 {
		sb.WriteString("\nTags:\n")
		for _, tag := range photo.Tags {
			sb.WriteString(fmt.Sprintf("- %s\n", tag.Title))
		}
	}

	return sb.String()
}

// --- Main Application Logic ---

// printVersion prints version information
func printVersion() {
	fmt.Printf("%s version %s\n", appName, version)
}

// printUsage prints a custom usage message
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "%s is a Model Context Protocol server that provides tools for accessing Unsplash photos.\n\n", appName)
	fmt.Fprintf(os.Stderr, "Requires the UNSPLASH_ACCESS_KEY environment variable to be set.\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

func main() {
	var transport string
	var port int
	var showVersion bool
	var showHelp bool

	// Override the default usage message
	flag.Usage = printUsage

	// Define command-line flags
	flag.StringVar(&transport, "transport", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or sse) (shorthand)")
	flag.IntVar(&port, "port", 8080, "Port for SSE transport")
	flag.IntVar(&port, "p", 8080, "Port for SSE transport (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&showVersion, "v", false, "Show version information and exit (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show this help message and exit")
	flag.BoolVar(&showHelp, "h", false, "Show this help message and exit (shorthand)")

	flag.Parse()

	// Handle version flag
	if showVersion {
		printVersion()
		os.Exit(0)
	}

	// Handle help flag
	if showHelp {
		printUsage()
		os.Exit(0)
	}

	// --- Get Unsplash Access Key ---
	accessKey := os.Getenv("UNSPLASH_ACCESS_KEY")
	if accessKey == "" {
		fmt.Fprintln(os.Stderr, "Error: UNSPLASH_ACCESS_KEY environment variable not set.")
		os.Exit(1)
	}

	// --- Create Unsplash Client ---
	unsplashClient, err := NewUnsplashClient(accessKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Unsplash client: %v", err)
		os.Exit(1)
	}

	// --- Create MCP Server ---
	s := server.NewMCPServer(
		appName,
		version,
		server.WithResourceCapabilities(false, false), // No resource providers needed
		server.WithLogging(),                          // Enable basic MCP logging
	)

	// --- Add Tools ---
	searchTool, searchHandler := createSearchPhotosTool(unsplashClient)
	s.AddTool(searchTool, searchHandler)

	getPhotoTool, getPhotoHandler := createGetPhotoTool(unsplashClient)
	s.AddTool(getPhotoTool, getPhotoHandler)

	randomPhotoTool, randomPhotoHandler := createRandomPhotoTool(unsplashClient)
	s.AddTool(randomPhotoTool, randomPhotoHandler)

	// --- Start Server based on transport ---
	fmt.Fprintf(os.Stderr, "%s v%s started successfully\n", appName, version)

	if transport == "stdio" {
		fmt.Fprintln(os.Stderr, "Unsplash MCP Server running on stdio")
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1) // Exit if server stops with error
		}
	} else if transport == "sse" {
		fmt.Fprintf(os.Stderr, "Unsplash MCP Server running on SSE mode at port %d\n", port)
		sseServer := server.NewSSEServer(s, server.WithBaseURL(fmt.Sprintf("http://localhost:%d", port)))
		log.Printf("Server started listening on :%d\n", port)
		if err := sseServer.Start(fmt.Sprintf(":%d", port)); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: Invalid transport type '%s'. Supported types: stdio, sse\n", transport)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Unsplash MCP Server finished.")
}
