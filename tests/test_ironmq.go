package main

import (
	"bufio"
	"github.com/iron-io/iron_mq_go"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func env() (err error) {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	b := bufio.NewReader(f)
	var line string

	for {
		line, err = b.ReadString('\n')
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}

		chunks := strings.SplitN(line, "=", 2) //split into 2 things
		os.Setenv(chunks[0], strings.TrimSpace(chunks[1]))
	}
	return
}

func poll(queue *ironmq.Queue, wait time.Duration) (out chan *ironmq.Message) {
	out = make(chan *ironmq.Message)
	go func() {
		for {
			msg, err := queue.Get()
			log.Println("Poll:", msg, err)
			switch err {
			case ironmq.EmptyQueue:
			case nil:
				out <- msg
			default:
				log.Println("queue error:", err)
			}
			<-time.After(wait)
		}
	}()
	return
}

func main() {
	os.Clearenv()
	if err := env(); err != nil {
		log.Fatal(err)
	}

	client := ironmq.NewClient(
		os.Getenv("IRON_MQ_PROJECT_ID"),
		os.Getenv("IRON_MQ_TOKEN"),
		ironmq.IronAWSUSEast,
	)
	queue := client.Queue("work_in")

	_, err := queue.Push(
		// `{"revisions":["e4ef402bacb2a4e0a86c0729ffd531e52eb68` +
		// 	`d52","34aa918aab43351e5ee86180cb170dc5b68f7a56"],"vc` +
		// 	`s":"git","repopath":"git://github.com/zeebo/irc","im` +
		// 	`portpath":"github.com/zeebo/irc","workspace":false}`,

		`{"revisions":["6d1ed8f9512102f30227ebfe8a327a572cbae` +
			`7f2","48d02e161b71b9ccce7d4e91439f43628f215003"],"vc` +
			`s":"git","repopath":"git://github.com/goods/starter"` +
			`,"importpath":"","workspace":true}`,
	)
	log.Println(err)
}
