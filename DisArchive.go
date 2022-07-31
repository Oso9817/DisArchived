package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	duper "projects/dupeCheck"
	megUpload "projects/megaUpload"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func downloadImage(url, fileName, dir string) error {
	response, err := http.Get(url)

	if err != nil {
		return err
	}
	//dirname, err := os.UserHomeDir()
	if err != nil {
		log.Println(err)
	}
	if response.StatusCode != 200 {
		return errors.New("Recieved NON-OK response code on url " + url)
	}
	defer response.Body.Close()

	file, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	if err != nil {
		return err
	}

	_, err = io.Copy(file, response.Body)

	if err != nil {
		return err
	}

	return nil
}

//put the last element of the 50 slice into a global variable then loop it in the message create event !start

func archive(s *discordgo.Session, lastChatID, channelID string, dir string) ([]string, error) {

	//index the first and last message, make the last message first to keep going 100 back
	var urlDl []string
	searchRange := 100
	message, err := s.ChannelMessages(channelID, searchRange, lastChatID, "", "")
	if err != nil {

		return urlDl, err
	}

	//c:\Users\Alonzo\Programming\DisArchive\DisArchive\images

	//gets last element in messages unique ID and makes it global
	foo := len(message) - 1
	if foo < 0 {
		err := errors.New("Out of range, last chatID: " + lastChatID)
		//log.Println()
		return urlDl, err
	}
	lastChatID = message[foo].ID

	//dirname, err := os.UserHomeDir()
	if err != nil {
		log.Println(err)
	}
	//look for last message in range, and go another 100 back
	for _, content := range message {

		for _, foo := range content.Attachments {
			if foo.Size <= 256 {
				continue
			}
			//anything less than 256 is probably an emote
			fileType := filepath.Ext(foo.Filename)
			fileName := foo.ID + fileType
			//create your own folder for images and place the path below
			//only creates if file does not exist, file use unique IDs names so it should not make duplicates
			if _, err := os.Stat(filepath.Join(dir, fileName)); !os.IsNotExist(err) {
				continue
			}

			log.Println("Creating file " + fileName + " " + foo.URL)

			err := downloadImage(foo.URL, fileName, dir)
			if err != nil {
				urlDL := append(urlDl, foo.URL)
				log.Println(urlDL)
				return urlDl, err

			}

		}

	}
	//separate archive function from downloading and just append to a a list to dl from later
	archive(s, lastChatID, channelID, dir)
	return urlDl, nil
}
func cmdArchive(imageDir, lastChatID, channelID string, s *discordgo.Session, m *discordgo.MessageCreate) error {
	//downloads all files sent in a chat server starting from a specific message ID backwards
	_, err := archive(s, lastChatID, channelID, imageDir)
	if err != nil {
		return err

	}
	return nil

}
func removeDupes(folder string) (int, error) {
	images, err := duper.Iterate(folder)
	if err != nil {
		return 0, err
	}
	hashes, err := duper.HashMap(folder, images)
	if err != nil {
		return 0, err
	}
	QTY := duper.HasDupes(hashes, folder)

	return QTY, err

}
func upload(folder string) error {
	err := megUpload.StartUpload(folder, "megaUpload/config.json", "Archived")
	if err != nil {
		return err
	}
	return nil

}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//change to whatever your directory is
	imageDir := "images2/"
	if strings.HasPrefix(m.Content, "!archive") {

		s.ChannelMessageSend(m.ChannelID, "Hol' up")
		//!start lastchatID channelID
		args := strings.SplitAfter(m.Content, " ")
		if len(args) != 3 {
			s.ChannelMessageSend(m.ChannelID, "Error parsing parameters, you seem to be missing some !start xxxx xxxx")
			return
		}
		err := cmdArchive(imageDir, args[1], args[2], s, m)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, "Error archiving, check log")
		}
		s.ChannelMessageSend(m.ChannelID, "Archive complete! Check directory")

	}
	if strings.HasPrefix(m.Content, "!dupey") {
		s.ChannelMessageSend(m.ChannelID, "Will remove duplicate photos now")
		QTY, err := removeDupes(imageDir)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, "Error removing duplicates, check log")
		}
		removedQTY := strconv.Itoa(QTY)
		s.ChannelMessageSend(m.ChannelID, "QTY of removed dupes: "+removedQTY)

	}
	if strings.HasPrefix(m.Content, "!upload") {
		s.ChannelMessageSend(m.ChannelID, "Will upload photos now")
		err := upload(imageDir)
		if err != nil {
			log.Println(err)
			s.ChannelMessageSend(m.ChannelID, "Error uploading files, check log")
		}
		s.ChannelMessageSend(m.ChannelID, "Upload complete")
	}

}
func main() {

	err := godotenv.Load("killerkeys.env")
	if err != nil {
		log.Fatal(err)
	}

	dkey := os.Getenv("DisKey")

	//checks if dir exists to store downloaded photos in
	//creates folder if not found
	_, err = os.Stat("images2/")
	if os.IsNotExist(err) {
		err = os.Mkdir("images2/", 0777)
		if err != nil {
			log.Println("Could not find nor create images/ folder")
		}
	}

	dg, err := discordgo.New("Bot " + dkey)
	if err != nil {
		log.Println(err)
	}

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAll)

	dg.State.MaxMessageCount = 50
	discordgo.NewState()

	err = dg.Open()

	if err != nil {
		log.Println(err)
		return
	}

	log.Println("CTRL-C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	defer dg.Close()
	//messageCreate()
}

/***
//optional function to zip files, not used in prod but could be used in future projects
func bigZip(filename string, files []string, dirname string) error {
	//log.Println(filename)
	newZip, err := os.Create("images\\" + filename)

	if err != nil {
		return err
	}

	defer newZip.Close()

	zipWriter := zip.NewWriter(newZip)
	defer zipWriter.Close()

	for _, file := range files {
		if !strings.HasSuffix(file, ".zip") {
			if err = AddFileToZip(zipWriter, file); err != nil {
				return err
			}
		}
	}
	return nil
}

//also not used due to nobody wanting to open a direct link to a zip file lol
func AddFileToZip(zipWriter *zip.Writer, filename string) error {

	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filename
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	log.Println(filename + "  --- Added to zip")
	return err
}
***/
