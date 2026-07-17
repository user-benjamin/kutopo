package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

//go:embed static
var staticFS embed.FS

func main() {
	port := flag.Int("port", 8090, "port to serve on (binds 127.0.0.1 only)")
	kubeContext := flag.String("context", "", "kubeconfig context to use (default: current context)")
	flag.Parse()

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: *kubeContext}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading kubeconfig: %v\n", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "building clientset: %v\n", err)
		os.Exit(1)
	}

	contextName := *kubeContext
	if contextName == "" {
		if raw, err := kubeConfig.RawConfig(); err == nil {
			contextName = raw.CurrentContext
		}
	}

	ctx := context.Background()
	store, err := NewStore(ctx, clientset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting informers: %v\n", err)
		os.Exit(1)
	}
	log.Printf("kutopo: syncing cluster state from context %q ...", contextName)
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := store.WaitForSync(syncCtx); err != nil {
		fmt.Fprintf(os.Stderr, "kutopo: %v\n", err)
		os.Exit(1)
	}
	log.Printf("kutopo: cache synced — steady-state API traffic is now watch-only")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/topology", func(w http.ResponseWriter, r *http.Request) {
		snap, err := store.Snapshot(contextName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(snap)
	})

	ui, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /", http.FileServerFS(ui))

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Printf("kutopo: context %q — serving http://%s", contextName, addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
