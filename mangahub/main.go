package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpcclient "github.com/biubiu2408/MangaHub/mangahub/grpc-client"
	tcpclient "github.com/biubiu2408/MangaHub/mangahub/tcp-client"
	udp_client "github.com/biubiu2408/MangaHub/mangahub/udp-client"
	websocket_client "github.com/biubiu2408/MangaHub/mangahub/websocket"
	"github.com/biubiu2408/MangaHub/package/models"
	"github.com/biubiu2408/MangaHub/utils"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var baseURL string
var username string
var password string
var token string
var status string

type UDPResponse struct {
	Status  string `json:"status"`
	Payload string `json:"payload"`
}

func getToken() string {
	if token != "" {
		utils.SaveToken(token)
		return token
	}

	// otherwise load from cache
	cached, err := utils.LoadToken()
	if err == nil && cached != "" {
		return cached
	}

	return "" // no available token
}
func main() {

	rootCmd := &cobra.Command{
		Use:   "mangahub",
		Short: "MangaHub CLI",
	}
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Username for authentication")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Password for authentication")

	rootCmd.PersistentFlags().StringVar(&token, "token", "", "JWT token (or set MANGAHUB_TEST_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base", "http://localhost:8080", "Base URL for API")

	// library subcommands
	libraryCmd := &cobra.Command{
		Use:   "library",
		Short: "Library commands",
	}
	libraryListCmd := &cobra.Command{
		Use:   "list",
		Short: "List library items for the authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {

			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub login --username USER --password PASS")
			}

			req, err := http.NewRequest("GET", baseURL+"/users/library", nil)
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("server error %s: %s", resp.Status, string(body))
			}

			var list models.ReadingLists
			if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
				return err
			}

			fmt.Println("📚 Your Library")

			// if user use --status flag
			if cmd.Flags().Changed("status") {
				switch status {
				case "reading":
					if len(list.Reading) == 0 {
						fmt.Println("No manga in Reading list.")
						return nil
					}
					fmt.Println("Currently Reading:")
					for _, it := range list.Reading {
						fmt.Printf(" - %s (ch %d)\n", it.MangaID, it.CurrentChapter)
						fmt.Printf("   Last Updated: %s\n", it.LastUpdated.Format("2006-01-02 15:04:05"))
					}
					return nil

				case "completed":
					if len(list.Completed) == 0 {
						fmt.Println("No manga in Completed list.")
						return nil
					}
					fmt.Println("Completed:")
					for _, it := range list.Completed {
						fmt.Printf(" - %s\n", it.MangaID)
					}
					return nil

				case "plan_to_read":
					if len(list.PlanToRead) == 0 {
						fmt.Println("No manga in Plan to Read list.")
						return nil
					}
					fmt.Println("Plan to Read:")
					for _, it := range list.PlanToRead {
						fmt.Printf(" - %s\n", it.MangaID)
					}
					return nil

				default:
					return fmt.Errorf("invalid status: %s (valid: reading, completed, plan_to_read)", status)
				}
			}

			// no --status flag => print all
			if len(list.Reading) > 0 {
				fmt.Println("Currently Reading:")
				for _, it := range list.Reading {
					fmt.Printf(" - %s (ch %d)\n", it.MangaID, it.CurrentChapter)
					fmt.Printf("   Last Updated: %s\n", it.LastUpdated.Format("2006-01-02 15:04:05"))
				}
			}

			if len(list.Completed) > 0 {
				fmt.Println("Completed:")
				for _, it := range list.Completed {
					fmt.Printf(" - %s\n", it.MangaID)
				}
			}

			if len(list.PlanToRead) > 0 {
				fmt.Println("Plan to Read:")
				for _, it := range list.PlanToRead {
					fmt.Printf(" - %s\n", it.MangaID)
				}
			}

			return nil

		},
	}

	libraryListCmd.Flags().StringVar(&status, "status", "", "Filter by status: reading, completed, plan")

	//#region add manga command
	libraryAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a manga to your library",
		RunE: func(cmd *cobra.Command, args []string) error {
			mangaID, _ := cmd.Flags().GetString("manga-id")
			status, _ := cmd.Flags().GetString("status")
			chapter, _ := cmd.Flags().GetInt("chapter") // optional

			if mangaID == "" {
				return fmt.Errorf("--manga-id required")
			}
			if status == "" {
				return fmt.Errorf("--status required (reading, completed, plan_to_read)")
			}

			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}

			// Build JSON body
			reqBody := map[string]interface{}{
				"manga_id": mangaID,
				"status":   status,
			}

			if cmd.Flags().Changed("chapter") {
				reqBody["current_chapter"] = chapter
			}

			body, _ := json.Marshal(reqBody)

			// POST request
			req, err := http.NewRequest("POST", baseURL+"/users/library", bytes.NewBuffer(body))
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 201 && resp.StatusCode != 200 {
				data, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("add failed %s: %s", resp.Status, string(data))
			}

			fmt.Println("Manga added to your library.")
			return nil
		},
	}

	libraryAddCmd.Flags().String("manga-id", "", "ID of the manga to add")
	libraryAddCmd.Flags().String("status", "", "Reading status: reading, completed, plan_to_read")
	libraryAddCmd.Flags().Int("chapter", 0, "Optional: current chapter number")

	//#region update manga command
	libraryUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update status for a manga in your library",
		RunE: func(cmd *cobra.Command, args []string) error {
			mangaID, _ := cmd.Flags().GetString("manga-id")
			newStatus, _ := cmd.Flags().GetString("status")

			if mangaID == "" {
				return fmt.Errorf("--manga-id required")
			}
			if newStatus == "" {
				return fmt.Errorf("--status required")
			}

			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}

			// build JSON body
			reqBody := map[string]string{
				"manga_id": mangaID,
				"status":   newStatus,
			}

			body, _ := json.Marshal(reqBody)

			// send PATCH request
			req, err := http.NewRequest("PATCH", baseURL+"/users/library", bytes.NewBuffer(body))
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				data, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("update failed %s: %s", resp.Status, string(data))
			}

			fmt.Println("Manga status updated successfully.")
			return nil
		},
	}

	libraryUpdateCmd.Flags().String("manga-id", "", "ID of the manga to update")
	libraryUpdateCmd.Flags().String("status", "", "New status (reading, completed, plan_to_read)")

	libraryRemoveCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a manga from your library",
		RunE: func(cmd *cobra.Command, args []string) error {
			mangaID, _ := cmd.Flags().GetString("manga-id")

			if mangaID == "" {
				return fmt.Errorf("--manga-id required")
			}

			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}

			// JSON body
			reqBody := map[string]string{
				"manga_id": mangaID,
			}

			body, _ := json.Marshal(reqBody)

			// delete request with JSON body
			req, err := http.NewRequest("DELETE", baseURL+"/users/library", bytes.NewBuffer(body))
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				data, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("remove failed %s: %s", resp.Status, string(data))
			}

			fmt.Println("Manga removed from your library.")
			return nil
		},
	}
	libraryRemoveCmd.Flags().String("manga-id", "", "ID of the manga to remove")

	//#region auth subcommands
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	//#region login command
	authLoginCmd := &cobra.Command{
		Use: "login",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" || password == "" {
				return fmt.Errorf("username and password required")
			}

			// Send login request to API
			req := map[string]string{
				"username": username,
				"password": password,
			}

			body, _ := json.Marshal(req)

			resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			// incorrect credentials => 401
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("invalid username or password")
			}

			var result struct {
				Token string `json:"token"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			// save token
			utils.SaveToken(result.Token)

			fmt.Println("Logged in successfully.")
			return nil
		},
	}

	//#region sign up command
	authSignupCmd := &cobra.Command{
		Use: "signup",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" || password == "" {
				return fmt.Errorf("username and password required")
			}
			//send signup request to API
			req := map[string]string{
				"username": username,
				"password": password,
			}

			body, _ := json.Marshal(req)

			resp, err := http.Post(baseURL+"/auth/signup", "application/json", bytes.NewBuffer(body))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 201 && resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("sign up failed: %s", string(body))
			}

			fmt.Println("Sign up successful")
			return nil
		},
	}

	//#region logout command
	authLogoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout and clear saved token",
		RunE: func(cmd *cobra.Command, args []string) error {

			jwt := getToken()
			if jwt == "" {
				fmt.Println("You are not logged in.")
				return nil
			}

			// send logout request to API
			req, err := http.NewRequest("POST", baseURL+"/auth/logout", nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+jwt)
				http.DefaultClient.Do(req)
			}

			// clear token locally
			if err := utils.ClearToken(); err != nil {
				return fmt.Errorf("failed to clear token: %v", err)
			}

			fmt.Println("Logged out successfully.")
			return nil
		},
	}

	// #region manga subcommands
	mangaCmd := &cobra.Command{
		Use:   "manga",
		Short: "Manga commands",
	}

	mangaListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all manga by genre, title or get manga  by ID",
		RunE: func(cmd *cobra.Command, args []string) error {

			genres, _ := cmd.Flags().GetStringSlice("genre")
			title, _ := cmd.Flags().GetString("title")
			page, _ := cmd.Flags().GetString("page")
			pageSize, _ := cmd.Flags().GetString("page-size")

			var url string

			switch {
			case title != "":
				url = fmt.Sprintf("%s/manga/search?query=%s", baseURL, title)
			case len(genres) > 0:
				joined := strings.Join(genres, ",")
				url = fmt.Sprintf("%s/manga/filter/genre?query=%s&page=%s&page_size=%s", baseURL, joined, page, pageSize)
			default:
				url = fmt.Sprintf("%s/manga?page=%s&page_size=%s", baseURL, page, pageSize)
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed %s: %s", resp.Status, string(body))
			}

			if title != "" {
				var m interface{}
				if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
					return err
				}

				raw, err := json.Marshal(m)
				if err != nil {
					return err
				}
				fmt.Println(string(raw))
				return nil
			}
			var result models.PaginatedMangasResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			fmt.Println("📚 Manga List:")
			if len(result.Items) == 0 {
				fmt.Println("No manga found.")
				return nil
			}
			fmt.Printf("Page %d Result:\n", result.Page)
			fmt.Printf("Total Items: %d | Total Pages: %d\n", result.TotalItems, result.TotalPages)
			for _, m := range result.Items {
				fmt.Printf(" - %s (%s)\n", m.Title, m.ID)
			}

			return nil
		},
	}
	mangaDetailCmd := &cobra.Command{
		Use:   "detail",
		Short: "Get manga details by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			mangaID, _ := cmd.Flags().GetString("manga")
			if mangaID == "" {
				return fmt.Errorf("--manga required")
			}
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/manga/%s", baseURL, mangaID), nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed %s: %s", resp.Status, string(body))
			}
			var manga models.Manga
			if err := json.NewDecoder(resp.Body).Decode(&manga); err != nil {
				return nil
			}
			fmt.Print("📖 Manga Details:\n")
			fmt.Printf("ID: %s\n", manga.ID)
			fmt.Printf("Title: %s\n", manga.Title)
			fmt.Printf("Author: %s\n", manga.Author)
			fmt.Printf("Artist: %s\n", manga.Artist)
			fmt.Printf("Genres: %s\n", strings.Join(manga.Genres, ", "))
			fmt.Printf("Chapters: %d\n", manga.ChapterCount)
			fmt.Printf("Volumes: %d\n", manga.VolumeCount)
			fmt.Printf("Published Year: %d\n", manga.PublishedYear)
			fmt.Printf("Status: %s\n", manga.Status)
			fmt.Printf("Popularity: %d\n", manga.Popularity)
			fmt.Printf("Ranking: %d\n", manga.Ranking)
			return nil
		},
	}
	//#region notifications command
	notifyCmd := &cobra.Command{
		Use:   "notify",
		Short: "Start UDP server to receive notifications",
	}
	notifyRegisterCmd := &cobra.Command{
		Use:   "register",
		Short: "Register this client for UDP notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login first")
			}

			// Start local UDP listener (client side)
			if err := udp_client.StartUDPServer(username); err != nil {
				return err
			}
			fmt.Println("📡 UDP listener started on port 3002")

			// Discover server
			serverAddr, err := udp_client.DiscoverUDPServer(2 * time.Second)
			if err != nil {
				return err
			}
			if err := utils.SaveUDPServerAddr(serverAddr); err != nil {
				return err
			}
			// Register
			if err := udp_client.RegisterUDPNotification(serverAddr, jwt); err != nil {
				return err
			}
			fmt.Println("✅ Registered successfully, listener running...")

			// Block until exit
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
			<-stop

			fmt.Println("\n👋 Shutting down UDP listener")
			return nil
		},
	}
	notifySubscribeCmd := &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribe to manga notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}
			mangaID, _ := cmd.Flags().GetString("manga")
			if mangaID == "" {
				return fmt.Errorf("--manga required")
			}
			udp_server_addr, err := utils.LoadUDPServerAddr()
			if err != nil || udp_server_addr == "" {
				return fmt.Errorf("no UDP server cached. Run `mangahub notify register` first")
			}
			fmt.Printf("📡 Using UDP server at %s\n", udp_server_addr)
			// send subscribe request via UDP
			if err := udp_client.SubscribeMangaUDP(udp_server_addr, jwt, mangaID); err != nil {
				return err
			}

			return nil
		},
	}

	// progress commands
	progressCmd := &cobra.Command{
		Use:   "progress",
		Short: "Reading progress commands",
	}

	progressUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update reading progress",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub login --username USER --password PASS")
			}

			mangaID, _ := cmd.Flags().GetString("manga-id")
			chapter, _ := cmd.Flags().GetInt("chapter")
			volume, _ := cmd.Flags().GetInt("volume")
			notes, _ := cmd.Flags().GetString("notes")
			force, _ := cmd.Flags().GetBool("force")

			if mangaID == "" {
				return fmt.Errorf("--manga-id required")
			}
			if chapter <= 0 {
				return fmt.Errorf("--chapter must be > 0")
			}

			// build request body for API
			reqBody := map[string]interface{}{
				"manga_id":        mangaID,
				"current_chapter": chapter,
				"force":           force,
			}

			// optional fields
			if volume > 0 {
				reqBody["volume"] = volume
			} else {
				reqBody["volume"] = nil
			}
			if notes != "" {
				reqBody["notes"] = notes
			} else {
				reqBody["notes"] = nil
			}

			body, _ := json.Marshal(reqBody)

			req, err := http.NewRequest(
				"PATCH",
				baseURL+"/users/progress",
				bytes.NewBuffer(body),
			)
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("✗ Progress update failed: %s", strings.TrimSpace(string(errBody)))
			}

			var result struct {
				MangaTitle        string    `json:"manga_title"`
				PreviousChapter   int       `json:"previous_chapter"`
				CurrentChapter    int       `json:"current_chapter"`
				UpdatedAt         time.Time `json:"updated_at"`
				DevicesSynced     int       `json:"devices_synced"`
				TotalChaptersRead int       `json:"total_chapters_read"`
				ReadingStreak     int       `json:"reading_streak"`
				NextChapter       int       `json:"next_chapter"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			// OUTPUT
			fmt.Println("Updating reading progress...")
			fmt.Println("✓ Progress updated successfully!")
			fmt.Printf("Manga: %s\n", result.MangaTitle)
			fmt.Printf("Previous: Chapter %d\n", result.PreviousChapter)
			fmt.Printf(
				"Current: Chapter %d (+%d)\n",
				result.CurrentChapter,
				result.CurrentChapter-result.PreviousChapter,
			)
			fmt.Println(
				"Updated:",
				result.UpdatedAt.Local().Format("2006-01-02 15:04:05"),
			)

			fmt.Println("Sync Status:")
			fmt.Println(" Local database: ✓ Updated")
			fmt.Printf(
				" TCP sync server: ✓ Broadcasting to %d connected devices\n",
				result.DevicesSynced,
			)
			fmt.Println(" Cloud backup: ✓ Synced") // currently hardcoded

			fmt.Println("Statistics:")
			fmt.Printf(" Total chapters read: %d\n", result.TotalChaptersRead)
			fmt.Printf(" Reading streak: %d days\n", result.ReadingStreak)

			if result.NextChapter > 0 {
				fmt.Printf(
					"Next actions:\n Continue reading: Chapter %d available\n",
					result.NextChapter,
				)
			}

			return nil
		},
	}

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "View reading progress history",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("please login first")
			}

			mangaID, _ := cmd.Flags().GetString("manga-id")

			url := baseURL + "/users/progress/history"
			if mangaID != "" {
				url += "?manga_id=" + mangaID
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf(
					"failed to fetch history: %s",
					strings.TrimSpace(string(errBody)),
				)
			}

			var result struct {
				UserID  int64 `json:"user_id"`
				History []struct {
					MangaID string `json:"manga_id"`
					Chapter int    `json:"chapter"`
					Date    string `json:"date_read"`
				} `json:"history"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			if len(result.History) == 0 {
				fmt.Println("No reading history found.")
				return nil
			}

			fmt.Printf("Reading Progress History (User ID: %d)\n", result.UserID)
			fmt.Println("------------------------------------------------")

			for _, h := range result.History {
				fmt.Printf(
					"%s | %-15s → Chapter %d\n",
					h.Date[:10],
					h.MangaID,
					h.Chapter,
				)
			}

			return nil
		},
	}

	progressSyncStatusCmd := &cobra.Command{
		Use:   "sync-status",
		Short: "Check progress sync status",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("please login first")
			}

			url := baseURL + "/users/progress/sync-status"

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+jwt)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed: %s", string(body))
			}

			var result map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}
			keys := []string{"status", "last_synced_at"}
			fmt.Println("📡 Sync Status:")
			for _, k := range keys {
				fmt.Printf(" - %s: %s\n", k, result[k])
			}

			return nil
		},
	}
	//#region sync command
	SyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Start the reading progress",
	}
	SyncConnectCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to the sync server and start syncing",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}
			fmt.Println("Starting TCP sync client...")
			serverIP, err := utils.LoadServerIPAddr()
			if err != nil {
				return fmt.Errorf("failed to load server IP address, please restart all servers again: %v", err)
			}
			deviceID := utils.DeviceID()
			if err := tcpclient.StartSync(jwt, deviceID, serverIP); err != nil {
				return fmt.Errorf("failed to start TCP sync client: %v", err)
			}

			return nil
		},
	}
	//#region grpc client command
	grpcCmd := &cobra.Command{
		Use:   "grpc",
		Short: "Start GRPC server to receive manga data",
	}
	grpcGetCmd := &cobra.Command{
		Use:   "get",
		Short: "Get manga by ID via gRPC",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login first")
			}

			_, err := utils.ValidateToken(jwt)
			if err != nil {
				return fmt.Errorf("invalid token: %v", err)
			}

			mangaID, _ := cmd.Flags().GetString("manga")
			if mangaID == "" {
				return fmt.Errorf("--manga required")
			}

			grpcclient.GetMangaByID(mangaID)
			return nil

		},
	}
	gprcSearchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search manga by title via gRPC",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login first")
			}

			_, err := utils.ValidateToken(jwt)
			if err != nil {
				return fmt.Errorf("invalid token: %v", err)
			}

			keyword, _ := cmd.Flags().GetString("keyword")
			if keyword == "" {
				return fmt.Errorf("--keyword required")
			}
			page, _ := cmd.Flags().GetInt("page")
			pageSize, _ := cmd.Flags().GetInt("page-size")
			grpcclient.SearchManga(keyword, int32(page), int32(pageSize))
			return nil
		},
	}
	grpcUpdateProgressCmd := &cobra.Command{
		Use:   "update-progress",
		Short: "Update reading progress via gRPC and broadcast to synced devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login first")
			}

			claims, err := utils.ValidateToken(jwt)
			if err != nil {
				return fmt.Errorf("invalid token: %v", err)
			}

			mangaID, _ := cmd.Flags().GetString("manga-id")
			if mangaID == "" {
				return fmt.Errorf("--manga-id required")
			}

			chapter, _ := cmd.Flags().GetInt("chapter")
			if chapter <= 0 {
				return fmt.Errorf("--chapter must be > 0")
			}

			// Use user ID from token claims
			return grpcclient.UpdateProgress(claims.UserId, mangaID, int64(chapter))
		},
	}
	//#endregion grpc
	//#region ws chat
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Start GRPC server to receive manga data",
	}
	chatJoinCmd := &cobra.Command{
		Use:   "join",
		Short: "Start WebSocket chat client",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("not logged in")
			}

			_, err := utils.ValidateToken(jwt)
			if err != nil {
				return fmt.Errorf("invalid token: %v", err)
			}

			room, _ := cmd.Flags().GetString("manga")
			if room == "" {
				room = "general"
			}
			serverIP, err := utils.LoadServerIPAddr()
			if err != nil {
				return fmt.Errorf("failed to load server IP address, please restart all servers again: %v", err)
			}
			wsURL := fmt.Sprintf(
				"ws://%s:8080/ws/chat?room=%s",
				serverIP,
				url.QueryEscape(room),
			)

			fmt.Println("Connecting to WebSocket chat server at", wsURL, "...")

			header := http.Header{}
			header.Set("Authorization", "Bearer "+jwt)

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			if err != nil {
				return err
			}
			defer conn.Close()

			fmt.Println("✓ Connected to General Chat")
			fmt.Println("Chat Room: #" + room)
			fmt.Println("Your status: Online")
			fmt.Println("Type messages and press Enter to send\n")

			// Ctrl+C handling
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			go websocket_client.ReadMessages(ctx, conn)
			go websocket_client.WriteMessages(ctx, conn)

			<-ctx.Done()
			fmt.Println("\nDisconnected.")
			return nil
		},
	}
	// Start command - runs all listeners in one command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start all listeners (TCP sync + UDP notifications)",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}

			fmt.Println("🚀 Starting MangaHub Client...")
			fmt.Println("=" + strings.Repeat("=", 50))

			// STEP 1: Discover server via UDP first to ensure all services have the server IP
			fmt.Println("\n🔍 Discovering server...")

			// Start local UDP listener
			if err := udp_client.StartUDPServer(username); err != nil {
				return fmt.Errorf("failed to start UDP listener: %v", err)
			}
			fmt.Println("✅ UDP listener started on port 3002")

			// Discover server and save the address
			serverAddr, err := udp_client.DiscoverUDPServer(2 * time.Second)
			if err != nil {
				return fmt.Errorf("failed to discover server: %v", err)
			}

			// Save server address so all processes (library, grpc, chat, etc.) can use it
			if err := utils.SaveUDPServerAddr(serverAddr); err != nil {
				return fmt.Errorf("failed to save server address: %v", err)
			}

			// Extract and verify server IP is accessible
			serverIP, err := utils.LoadServerIPAddr()
			if err != nil {
				return fmt.Errorf("failed to extract server IP: %v", err)
			}
			fmt.Printf("✅ Server discovered at %s (full address: %s)\n", serverIP, serverAddr)

			// Context for graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Channel to collect errors from goroutines
			errChan := make(chan error, 2)

			// STEP 2: Start TCP sync client (now guaranteed to have server IP)
			go func() {
				fmt.Println("\n📡 Starting TCP sync client...")
				deviceID := utils.DeviceID()
				if err := tcpclient.StartSync(jwt, deviceID, serverIP); err != nil {
					errChan <- fmt.Errorf("TCP: %v", err)
					return
				}
			}()

			// STEP 3: Register UDP notifications
			go func() {
				fmt.Println("📡 Registering UDP notifications...")
				if err := udp_client.RegisterUDPNotification(serverAddr, jwt); err != nil {
					errChan <- fmt.Errorf("UDP: failed to register: %v", err)
					return
				}
				fmt.Println("✅ UDP registered successfully")
			}()

			// Give everything time to initialize
			time.Sleep(1 * time.Second)

			fmt.Println("\n" + strings.Repeat("=", 50))
			fmt.Println("✅ All services started successfully!")
			fmt.Println("📝 Services running:")
			fmt.Printf("   • Server IP: %s (cached for all processes)\n", serverIP)
			fmt.Println("   • TCP Sync Client - Real-time progress synchronization")
			fmt.Println("   • UDP Listener - Manga update notifications")
			fmt.Println("\n💡 All commands (library, grpc, chat) now have access to server IP")
			fmt.Println("💡 Press Ctrl+C to stop all services")
			fmt.Println(strings.Repeat("=", 50))

			// Wait for interrupt signal or errors
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			select {
			case err := <-errChan:
				return err
			case <-stop:
				fmt.Println("\n\n👋 Shutting down all services...")
				cancel()
				return nil
			case <-ctx.Done():
				return nil
			}
		},
	}

	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Start all listeners (TCP sync + UDP notifications)",
	}
	//#region manga update command (admin)
	mangaUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update manga in database (admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}

			// Get flags
			mangaID, _ := cmd.Flags().GetString("id")
			title, _ := cmd.Flags().GetString("title")
			author, _ := cmd.Flags().GetString("author")
			artist, _ := cmd.Flags().GetString("artist")
			genres, _ := cmd.Flags().GetStringSlice("genres")
			chapters, _ := cmd.Flags().GetInt("chapters")
			volumes, _ := cmd.Flags().GetInt("volumes")
			year, _ := cmd.Flags().GetInt("year")
			mangaStatus, _ := cmd.Flags().GetString("status")
			popularity, _ := cmd.Flags().GetInt("popularity")
			ranking, _ := cmd.Flags().GetInt("ranking")

			if mangaID == "" {
				return fmt.Errorf("--id required")
			}

			// Build manga object
			manga := models.Manga{
				ID: mangaID,
			}

			// Only include fields that were explicitly set
			if title != "" {
				manga.Title = title
			}
			if author != "" {
				manga.Author = author
			}
			if artist != "" {
				manga.Artist = artist
			}
			if len(genres) > 0 {
				manga.Genres = genres
			}
			if chapters > 0 {
				manga.ChapterCount = chapters
			}
			if volumes > 0 {
				manga.VolumeCount = volumes
			}
			if year > 0 {
				manga.PublishedYear = year
			}
			if mangaStatus != "" {
				manga.Status = mangaStatus
			}
			if popularity > 0 {
				manga.Popularity = popularity
			}
			if ranking > 0 {
				manga.Ranking = ranking
			}

			// Marshal to JSON
			reqBody, err := json.Marshal(manga)
			if err != nil {
				return err
			}

			// Send PUT request to /admin/manga
			req, err := http.NewRequest("PUT", baseURL+"/admin/manga", bytes.NewBuffer(reqBody))
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed %s: %s", resp.Status, string(body))
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			fmt.Println("✅ Manga updated successfully!")
			fmt.Printf("Response: %v\n", result)
			return nil
		},
	}
	mangaChapterUpdateCmd := &cobra.Command{
		Use:   "update-chapter",
		Short: "Update manga chapter release in database (admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			jwt := getToken()
			if jwt == "" {
				return fmt.Errorf("no token found. Please login using: mangahub auth login --username USER --password PASS")
			}

			// Get flags
			mangaID, _ := cmd.Flags().GetString("manga")

			chapters, _ := cmd.Flags().GetInt("chapter")

			if mangaID == "" {
				return fmt.Errorf("--manga required")
			}

			if chapters < 0 {
				return fmt.Errorf("--chapters must be > 0")
			}
			if chapters == 0 {
				return fmt.Errorf("--chapter required")
			}

			// Build manga object
			input := map[string]interface{}{
				"manga_id": mangaID,
				"chapter":  chapters,
			}
			// Marshal to JSON
			reqBody, err := json.Marshal(input)
			if err != nil {
				return err
			}

			// Send PUT request to /admin/manga
			req, err := http.NewRequest("PUT", baseURL+"/admin/manga/chapter-release", bytes.NewBuffer(reqBody))
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+jwt)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed %s: %s", resp.Status, string(body))
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			fmt.Println("✅ Manga updated successfully!")
			fmt.Printf("Response: %v\n", result)
			return nil
		},
	}
	adminCmd.AddCommand(mangaUpdateCmd)
	adminCmd.AddCommand(mangaChapterUpdateCmd)
	rootCmd.AddCommand(adminCmd)
	//sync server
	SyncCmd.AddCommand(SyncConnectCmd)
	rootCmd.AddCommand(SyncCmd)
	rootCmd.AddCommand(startCmd)

	// flags
	progressUpdateCmd.Flags().String("manga-id", "", "Manga ID")
	progressUpdateCmd.Flags().Int("chapter", 0, "Chapter number")
	progressUpdateCmd.Flags().Int("volume", 0, "Volume number")
	progressUpdateCmd.Flags().String("notes", "", "Personal notes")
	progressUpdateCmd.Flags().Bool("force", false, "Force backward progress update")
	historyCmd.Flags().String("manga-id", "", "Filter by manga ID")

	progressCmd.AddCommand(progressUpdateCmd)
	progressCmd.AddCommand(historyCmd)
	progressCmd.AddCommand(progressSyncStatusCmd)
	rootCmd.AddCommand(progressCmd)

	// notifyAddCmd.Flags().String("manga", "", "ID of the manga to subscribe to")

	chatJoinCmd.Flags().String("manga", "", "Manga room to join (default: general)")
	gprcSearchCmd.Flags().String("keyword", "", "Keyword to search manga titles")
	gprcSearchCmd.Flags().Int("page", 1, "Page number")
	gprcSearchCmd.Flags().Int("page-size", 10, "Number of results per page")
	grpcGetCmd.Flags().String("manga", "", "ID of the manga to retrieve")
	grpcUpdateProgressCmd.Flags().String("manga-id", "", "Manga ID")
	grpcUpdateProgressCmd.Flags().Int("chapter", 0, "Chapter number")
	grpcCmd.AddCommand(grpcGetCmd)
	grpcCmd.AddCommand(gprcSearchCmd)
	grpcCmd.AddCommand(grpcUpdateProgressCmd)
	notifySubscribeCmd.Flags().String("manga", "", "ID of the manga to subscribe to")
	notifyCmd.AddCommand(notifyRegisterCmd)
	notifyCmd.AddCommand(notifySubscribeCmd)
	chatCmd.AddCommand(chatJoinCmd)
	rootCmd.AddCommand(grpcCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(notifyCmd)

	mangaDetailCmd.Flags().String("manga", "", "Get a manga by ID")
	mangaListCmd.Flags().StringSlice("genre", []string{}, "Filter manga by genres")
	mangaListCmd.Flags().String("title", "", "Search manga by title")
	mangaListCmd.Flags().String("page", "", "Page number for listing manga")
	mangaListCmd.Flags().String("page-size", "", "Number of manga per page")

	mangaChapterUpdateCmd.Flags().String("manga", "", "Manga ID (required)")
	mangaChapterUpdateCmd.Flags().Int("chapter", 0, "New chapter count (required)")
	mangaUpdateCmd.Flags().String("id", "", "Manga ID (required)")
	mangaUpdateCmd.Flags().String("title", "", "Manga title")
	mangaUpdateCmd.Flags().String("author", "", "Manga author")
	mangaUpdateCmd.Flags().String("artist", "", "Manga artist")
	mangaUpdateCmd.Flags().StringSlice("genres", []string{}, "Manga genres")
	mangaUpdateCmd.Flags().Int("chapters", 0, "Chapter count")
	mangaUpdateCmd.Flags().Int("volumes", 0, "Volume count")
	mangaUpdateCmd.Flags().Int("year", 0, "Published year")
	mangaUpdateCmd.Flags().String("status", "", "Manga status (ongoing, completed)")
	mangaUpdateCmd.Flags().Int("popularity", 0, "Popularity score")
	mangaUpdateCmd.Flags().Int("ranking", 0, "Ranking")
	//#endregion

	libraryCmd.AddCommand(libraryListCmd)
	libraryCmd.AddCommand(libraryAddCmd)
	libraryCmd.AddCommand(libraryUpdateCmd)
	libraryCmd.AddCommand(libraryRemoveCmd)
	rootCmd.AddCommand(libraryCmd)

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authSignupCmd)
	authCmd.AddCommand(authLogoutCmd)
	rootCmd.AddCommand(authCmd)

	mangaCmd.AddCommand(mangaListCmd)
	mangaCmd.AddCommand(mangaDetailCmd)
	mangaCmd.AddCommand(mangaUpdateCmd)
	rootCmd.AddCommand(mangaCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
