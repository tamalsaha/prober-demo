package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux" // need to use dep for package management
	"github.com/spf13/cobra"
)

func NewCmdRunClient() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run-client",
		Short: "run client where probes will be executed",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Running... client")
			return runClient()
		},
	}
	return cmd
}
func runClient() error {
	var wg sync.WaitGroup

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Starting HTTP Server")
	wg.Add(1)
	go runHttpServer(&wg, done)

	fmt.Println("Starting TCP Client")
	wg.Add(1)
	go runTCPServer(&wg, done)

	wg.Wait()

	fmt.Println("Exiting Client")

	return nil
}

func runHttpServer(wg *sync.WaitGroup, done chan os.Signal) {
	defer wg.Done()

	router := mux.NewRouter()
	router.HandleFunc("/", httpGETHandler).Methods("GET")
	router.HandleFunc("/success", httpGETHandler).Methods("GET")
	router.HandleFunc("/fail", httpGETHandler).Methods("GET")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Print("Server Started")

	<-done
	log.Print("Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		// extra handling here
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Println("server Shutdown Failed:%+v", err)
		return
	}
	log.Print("Server Exited Properly")
}

func httpGETHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("============== Received request")
	fmt.Println(r.URL.Path)
	switch r.URL.Path {
	case "/success":
		fmt.Println("Request in path: /success")
		w.WriteHeader(http.StatusOK)
	case "/fail":
		fmt.Println("Request in path: /fail")
		w.WriteHeader(http.StatusForbidden)
	}
}

func runTCPServer(wg *sync.WaitGroup, done chan os.Signal) {
	defer wg.Done()
	listener, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		log.Fatal("tcp server listener error:", err)
	}

	fmt.Println("Starting TCP server...........")
	var wg2 sync.WaitGroup
	go func(done chan os.Signal) {
		<-done
		fmt.Println("Stop signal recieved. Stopping TCP server.............")
		listener.Close()
		wg2.Wait()
		return
	}(done)
	for {
		fmt.Println("listening.....")
		conn, err := listener.Accept()
		fmt.Println("new request..............")
		if err != nil {
			fmt.Println("tcp server accept error", err)
			os.Exit(1)
		}
		wg2.Add(1)
		go handleConnection(&wg2, conn)
	}
}

func handleConnection(wg2 *sync.WaitGroup, conn net.Conn) {
	fmt.Println("Handling Request.....")
	defer wg2.Done()
	defer conn.Close()

	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	// Send a response back to person contacting us.
	conn.Write([]byte("Message received."))
	// Close the connection when you're done with it.
	fmt.Println("Request Handling Done. Closing.....")
	//return
}
