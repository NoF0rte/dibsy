package bot

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gocolly/colly"
	"github.com/robfig/cron/v3"
)

type dibbedElement struct {
	Attr map[string]string
	Text string
}

type Dibsy struct {
	discord       *discordgo.Session
	config        *Config
	cron          *cron.Cron
	cronIDsByDibs map[Dib]cron.EntryID
	baselineResp  map[Dib]string
}

func New(config *Config) (*Dibsy, error) {
	discord, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		return nil, err
	}

	dibsy := &Dibsy{
		discord:       discord,
		config:        config,
		cron:          cron.New(),
		cronIDsByDibs: make(map[Dib]cron.EntryID),
		baselineResp:  make(map[Dib]string),
	}

	for _, dib := range config.Dibs {
		err := dibsy.schedule(dib)
		if err != nil {
			return nil, err
		}
	}

	return dibsy, nil
}

func (d *Dibsy) Start() error {
	log.Println("Starting dibsy...")
	err := d.discord.Open()
	if err != nil {
		return err
	}
	d.cron.Start()
	return nil
}

func (d *Dibsy) Close() {
	log.Println("Stopping dibsy...")
	d.discord.Close()
	ctx := d.cron.Stop()
	<-ctx.Done()
}

func (d *Dibsy) schedule(dib Dib) error {
	entry, err := d.cron.AddFunc(fmt.Sprintf(`@every %s`, dib.Interval), func() {
		log.Printf("Executing dib: %s\n", dib.Name)
		d.exec(dib)
	})

	if err != nil {
		return err
	}

	d.cronIDsByDibs[dib] = entry

	if dib.Type == DibDiff {
		d.execDiff(dib)
	}

	return nil
}

func (d *Dibsy) remove(dib Dib) {
	id, exists := d.cronIDsByDibs[dib]
	if !exists {
		return
	}

	d.cron.Remove(id)
}

func (d *Dibsy) exec(dib Dib) {
	if dib.Type == DibDiff {
		d.execDiff(dib)
	} else {
		d.execHTML(dib)
	}
}

func (d *Dibsy) success(dib Dib) {
	message := fmt.Sprintf("New Dib!\n%s\n\n%s", dib.Message, dib.URL)
	_, err := d.discord.ChannelMessageSend(d.config.DiscordNotifyChannel, message)
	if err != nil {
		log.Println(err)
	}
	d.remove(dib)
}

func (d *Dibsy) execHTML(dib Dib) {
	collector := colly.NewCollector()
	collector.OnHTML(dib.Selector, func(h *colly.HTMLElement) {
		element := dibbedElement{
			Attr: make(map[string]string),
			Text: h.Text,
		}
		for _, node := range h.DOM.Nodes {
			for _, attr := range node.Attr {
				element.Attr[attr.Key] = attr.Val
			}
		}

		funcMap := make(template.FuncMap)
		funcMap["ieq"] = func(s1, s2 string) bool {
			return strings.EqualFold(s1, s2)
		}

		t, err := template.New("dib").Funcs(funcMap).Parse(fmt.Sprintf(`{{if %s}}true{{end}}`, dib.Condition))
		if err != nil {
			log.Println(err)
			return
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, element)
		if err != nil {
			log.Println(err)
			return
		}

		if buf.String() == "" {
			return
		}

		d.success(dib)
	})
	collector.Visit(dib.URL)
}

func (d *Dibsy) get(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (d *Dibsy) execDiff(dib Dib) {
	baseline, ok := d.baselineResp[dib]

	body, err := d.get(dib.URL)
	if err != nil {
		if !ok {
			log.Printf("Error getting baseline: %v", err)
		} else {
			log.Printf("Error sending request: %v", err)
		}
		return
	}

	if !ok {
		d.baselineResp[dib] = body
		return
	}

	if body != baseline {
		d.success(dib)
	}
}
