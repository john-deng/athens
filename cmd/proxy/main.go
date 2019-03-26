package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gomods/athens/cmd/proxy/actions"
	"github.com/gomods/athens/pkg/build"
	"github.com/gomods/athens/pkg/config"
)

var (
	configFile = flag.String("config_file", "", "The path to the config file")
	version    = flag.Bool("version", false, "Print version information and exit")
	replaceFile = flag.String("replace_file", "./replace.cfg", "The path to the mod file")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Println(build.String())
		os.Exit(0)
	}
	conf, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("could not load config file: %v", err)
	}

	handler, err := actions.App(conf)
	if err != nil {
		log.Fatal(err)
	}

	cert, key, err := conf.TLSCertFiles()
	if err != nil {
		log.Fatal(err)
	}
	_,err=os.Stat(*replaceFile)
	if err == nil{
		f, err := os.Open(*replaceFile)
		if err == nil {
			fmt.Println("load mod convert file")
			config.ModMap=make(map[string]string)
			buf := bufio.NewReader(f)
			for {
				line, err := buf.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
				}
				line = strings.TrimSpace(line)
				str:=strings.SplitN(line,"######",2)
				config.ModMap[str[0]]=str[1]
			}
		}
	}
	srv := &http.Server{
		Addr:    conf.Port,
		Handler: handler,
	}
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("Starting application at port %v", conf.Port)
	if cert != "" && key != "" {
		err = srv.ListenAndServeTLS(conf.TLSCertFile, conf.TLSKeyFile)
	} else {
		err = srv.ListenAndServe()
	}

	if err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-idleConnsClosed
}
