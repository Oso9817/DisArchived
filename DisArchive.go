package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	zaar "projects/megaUpload"
	//"strconv"

	//"strcov"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	//"google.golang.org/api/file/v1"
)

func downloadImage(url, fileName string) error {
	response, err := http.Get(url)

	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return errors.New("Recieved NON-OK response code on url " + url)
	}
	defer response.Body.Close()

	file, err := os.Create("c:\\Users\\Alonzo\\Programming\\DisArchived\\DisArchived\\images\\" + fileName)
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
var (
	saveLocation = "c:\\Users\\Alonzo\\Programming\\DisArchived\\DisArchived\\images\\"
)

func archive(s *discordgo.Session, lastChatID, channelID string) ([]string, error) {

	//index the first and last message, make the last message first to keep going 100 back
	var urlDl []string
	searchRange := 100
	message, err := s.ChannelMessages(channelID, searchRange, lastChatID, "", "")
	if err != nil {

		return urlDl, err
	}

	//c:\Users\Alonzo\Programming\DisArchive\DisArchive\images
	//var messageID []int
	//gets last element in messages unique ID and makes it global
	foo := len(message) - 1
	if foo < 0 {
		err := errors.New("Out of range, last chatID: " + lastChatID)
		//log.Println()
		return urlDl, err
	}
	lastChatID = message[foo].ID
	//look for last message in range, and go another 100 back
	for _, content := range message {

		if len(content.Attachments) != 0 {

			for _, foo := range content.Attachments {
				if foo.Size >= 256 {
					fileType := strings.SplitAfter(foo.Filename, ".")
					fileName := foo.ID + "." + fileType[1]
					//create your own folder for images and place the path below
					//only creates if file does not exist, file use unique IDs names so it should not make duplicates
					if _, err := os.Stat(saveLocation + fileName); os.IsNotExist(err) {

						log.Println("Creating file " + fileName + " " + foo.URL)

						err := downloadImage(foo.URL, fileName)
						if err != nil {
							urlDL := append(urlDl, foo.URL)
							log.Println(urlDL)
							return urlDl, err

						}
					}

				}

			}

		}

	}
	//separate archive function from downloading and just append to a a list to dl from later
	archive(s, lastChatID, channelID)
	return urlDl, nil
}
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!start") {
		//var filename []string
		zipName := "photos.zip"
		s.ChannelMessageSend(m.ChannelID, "Hol' up")
		//!start lastchatID channelID
		//args := strings.SplitAfter(m.Content, " ")
		//last chat ID 176595202172125185 images in the 172,000's
		//searches thru channel that the command was sent in
		/***
		_, err := archive(s, args[1], args[2])
		if err != nil {
			log.Println(err)
		}
		s.ChannelMessageSend(m.ChannelID, "Done! check directory location")


		file, err := os.Open(saveLocation)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()

		fileList, _ := file.Readdir(0)

		for _, files := range fileList {
			filename = append(filename, "images/"+files.Name())
		}
		//log.Println(filename)

		err = bigZip("photos.zip", filename)
		if err != nil {
			log.Println(err)
		}
		s.ChannelMessageSend(m.ChannelID, "Zip saved!")
		***/
		url, err := zaar.StartUpload(zipName)
		if err != nil {
			log.Println(err)
		}
		s.ChannelMessageSend(m.ChannelID, url)
	}
}
func main() {
	err := godotenv.Load("C:/Users/Alonzo/Programming/Go-Rito/isHeBoosted/killerkeys.env")
	if err != nil {
		log.Fatal(err)
	}
	dkey := os.Getenv("DisKey")
	dg, err := discordgo.New("Bot " + dkey)

	//log.Println(reflect.TypeOf(dg))
	if err != nil {
		fmt.Println(err)
		return
	}
	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAll)

	dg.State.MaxMessageCount = 50
	discordgo.NewState()

	err1 := dg.Open()

	if err1 != nil {
		fmt.Println(err1)
		return
	}

	fmt.Println("CTRL-C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	defer dg.Close()
	//messageCreate()
}

func bigZip(filename string, files []string) error {
	//log.Println(filename)
	newZip, err := os.Create(saveLocation + filename)

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
	return err
}
