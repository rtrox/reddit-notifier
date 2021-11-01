package main

import (
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type Author struct {
	Name    string `json:"name,omitempty"`
	Url     string `json:"url,omitempty"`
	IconUrl string `json:"icon_url,omitempty"`
}

type URL struct {
	URL string `json:"url,omitempty"`
}

type Footer struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type Embed struct {
	Author Author `json:"author,omitempty"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
	Fields      []Field `json:"fields"`
	Thumbnail URL `json:"thumbnail,omitempty"`
	Image URL `json:"image,omitempty"`
	Footer Footer `json:"footer,omitempty"`
}

type WebhookRequest struct {
	Username  string `json:"username,omitempty"`
	AvatarUrl string `json:"avatar_url,omitempty"`
	Content   string `json:"content,omitempty"`
	Embeds    []Embed  `json:"embeds,omitempty"`
}

var notifiedIds = make(map[string]bool)


func init() {
	log.SetFormatter(&log.TextFormatter{})

	log.SetOutput(os.Stderr)

	log.SetLevel(log.InfoLevel)
}

func main() {
	subreddit := os.Getenv("REDDIT_SUBREDDIT")
	if subreddit == "" {
		log.Fatalf("REDDIT_SUBREDDIT environmental variable not set.")
	}

	expr := os.Getenv("REDDIT_MATCHER")
	if expr == "" {
		log.Fatalf("REDDIT_MATCHER environmental variable not set.")
	}
	matcher, err := regexp.Compile(expr)

	discordUrl := os.Getenv("REDDIT_DISCORD_URL")
	if discordUrl == "" {
		log.Fatalf("REDDIT_DISCORD_URL environmental variable not set.")
	}
	if err != nil {
		log.Fatalf("could not compile regex for matcher: %v", err)
	}

	log.Info("reddit-notifier initialized.")

	sig := make(chan os.Signal, 1)
	defer close(sig)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	posts, errs, stop := reddit.DefaultClient().Stream.Posts(subreddit, reddit.StreamInterval(time.Second*3))
	defer stop()

	for {
		select {
		case post, ok := <-posts:
			if !ok {
				return
			}
			if shouldNotifyPost(post, matcher) {
				if err := notifyPost(post, discordUrl); err != nil {
					log.Errorf("Error notifying for post: %v", err)
				}
			}
		case err, ok := <-errs:
			if !ok {
				return
			}
			log.Errorf("Error streaming posts: %v", err)
		case rcvSig, ok := <-sig:
			if !ok {
				return
			}
			log.Infof("Stopping due to %s signal.\n", rcvSig)
			return
		}
	}
}

func shouldNotifyPost(post *reddit.Post, matcher *regexp.Regexp) bool {
	if _, ok := notifiedIds[post.ID]; ok {
		log.Debugf("Filtering out %s as it's already notified.", post.Title)
		return false
	}

	if post.LinkFlairText == "NO MORE INVITES" {
		log.Debugf("Filtering out %s based on NO MORE INVITES flair.", post.Title)
		return false
	}

	if post.LinkFlairText == "WANTED" {
		log.Debugf("Filtering out %s based on WANTED flair.", post.Title)
		return false
	}

	if !matcher.MatchString(post.Title) {
		log.Debugf("Filtering out %s based on regexp matcher.", post.Title)
		return false
	}

	return true
}


func notifyPost(post *reddit.Post, url string) error {
	log.Infof("Notifying post: %s, flair: %s\n", post.Title, post.LinkFlairText)

	notif := WebhookRequest{
		Content: "New Usenet Invite Post matching your filters:",
		Embeds: []Embed{
			Embed{
				Title: post.Title,
				URL:   post.URL,
				Description: post.Body,
				Fields: []Field{
					Field{
						Name: "Author",
						Value: fmt.Sprintf("[%s](https://www.reddit.com/message/compose/?to=%s)", post.Author, post.Author),
						Inline: true,
					},
					Field{
						Name: "Flair",
						Value: post.LinkFlairText,
						Inline: true,
					},
				},
			},
		},
	}
	body, _ := json.Marshal(notif)

	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode != 204 {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
		}).Errorf("Webhook Failed, Response Body: %s", string(body))
	}

	return nil
}
