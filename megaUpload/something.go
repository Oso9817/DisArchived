package megaUpload

import (
	//"errors"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	//"path"
	"strings"
	"sync"
	"time"

	//"github.com/sqs/goreturns/returns"
	"github.com/t3rm1n4l/go-mega"
	//"os/user"
	//"path"
)

var (
	config = flag.String("wfawef", CONFIG_FILE, "Config file path")
)

const (
	CONFIG_FILE = "C:/Users/Alonzo/Programming/DisArchived/DisArchived/megaUpload/config.json"
)

func StartUpload() error {

	conf := new(Config)
	err := conf.Parse(*config)
	if err != nil {
		return err
	}

	client, err := NewMegaClient(conf)
	if err != nil {
		return err
	}
	err = client.Login()
	if err != nil {
		return err
	}

	arg1 := "C:/Users/Alonzo/Programming/DisArchived/DisArchived/images/"
	arg2 := "mega:/"

	//reads directory and generates list of contents
	files, err := ioutil.ReadDir(arg1)
	if err != nil {
		log.Fatal(err)
	}
	//loops through previously generated lists and uploads them individually, does not crash if file already exists as it is possible it does
	for _, file := range files {

		err = client.Put(arg1, file.Name(), arg2)
		if err != ErrFileExist && err != nil {
			log.Printf("ERROR: Uploading %s to %s failed: (%s)", arg1+file.Name(), arg2, err)

		} else if err == ErrFileExist {
			log.Println(file.Name() + " -- Already exists in destination")
		}

	}

	return err
}

type Config struct {
	BaseUrl         string
	Retries         int
	DownloadWorkers int
	UploadWorkers   int
	TimeOut         int
	User            string
	Password        string
	Recursive       bool
	Force           bool
	SkipSameSize    bool
	SkipError       bool
	Verbose         int
}
type MegaClient struct {
	cfg  *Config
	mega *mega.Mega
}

func (mc *MegaClient) getNode(dstres string) ([]*mega.Node, error) {
	var nodes []*mega.Node
	var node *mega.Node

	root, pathsplit, err := getLookupParams(dstres, mc.mega.FS)
	if err != nil {
		return nil, err
	}

	if len(*pathsplit) > 0 {
		nodes, err = mc.mega.FS.PathLookup(root, *pathsplit)
	}

	if err != nil && err != mega.ENOENT {
		return nil, err
	}

	lp := len(*pathsplit)
	ln := len(nodes)

	//	var name string
	switch {
	case lp == ln:
		if lp == 0 {
			node = root
		} else {
			//goes here
			node = nodes[ln-1]
			/***
			if node.GetType() == mega.FOLDER {
				if !strings.HasSuffix(dstres, "/") {
					return nil, ErrDirExist
				}
			} else {
				if strings.HasSuffix(dstres, "/") {
					return nil, ErrNonDir
				}
				if len(nodes) > 1 {
					node = nodes[ln-2]

				} else {
					node = root
				}
			}***/
		}
	case ln == 0 && lp == 1:
		if !strings.HasSuffix(dstres, "/") {
			node = root
		} else {
			return nil, err
		}
	default:
		return nil, err
	}

	nodes, err = mc.mega.FS.GetChildren(node)
	if err != nil {
		return nil, err
	}
	return nodes, err

}

func (mc *MegaClient) Put(srcpath, name, dstres string) error {
	//var nodes []*mega.Node

	var node *mega.Node
	srcpath = srcpath + name

	info, err := os.Stat(srcpath)

	if err != nil {
		return ErrSrc
	}
	if info.Mode()&os.ModeType != 0 {
		return ErrNotFile
	}

	root, _, err := getLookupParams(dstres, mc.mega.FS)
	if err != nil {
		return err
	}

	var ch *chan int
	var wg sync.WaitGroup
	var bar []string
	bar = append(bar, "Personal")
	war := mc.mega.FS.GetRoot()
	//checks main mega children if folder exists,

	//	creates it if not
	query, err := mc.mega.FS.PathLookup(war, bar)
	if err == ErrNoFolder {
		node, err := mc.mega.CreateDir("Personal", root)
		if err != nil {
			return err
		}
		_, err = mc.mega.UploadFile(srcpath, node, name, ch)
		if err != nil {
			//crashes here
			return err
		}
		log.Println(srcpath + " -- Succesfully uploaded to destination")
		return err

	}
	if err != nil && err != ErrNoFolder {
		return err
	}

	node = query[0]
	children, _ := mc.mega.FS.GetChildren(node)
	for _, c := range children {
		if c.GetName() == name {
			return ErrFileExist
		}
	}

	_, err = mc.mega.UploadFile(srcpath, node, name, ch)
	if err != nil {
		//crashes here
		return err
	}
	log.Println(srcpath + " -- Succesfully uploaded to destination")

	wg.Wait()
	return err
}

var (
	ErrConfig    = errors.New("invalid json config")
	ErrMegaPath  = errors.New("invalid mega path")
	ErrNotFile   = errors.New("requested object is not a file")
	ErrDest      = errors.New("invalid destination path")
	ErrSrc       = errors.New("invalid source path")
	ErrSync      = errors.New("invalid sync command parameters")
	ErrNonDir    = errors.New("a non-directory exists at this path")
	ErrFileExist = errors.New("file with same name already exists")
	ErrDirExist  = errors.New("a directory with same name already exists")
	ErrNoFolder  = mega.ENOENT
)

func getLookupParams(resource string, fs *mega.MegaFS) (*mega.Node, *[]string, error) {
	resource = strings.TrimSpace(resource)
	args := strings.SplitN(resource, ":", 2)
	if len(args) != 2 || !strings.HasPrefix(args[1], "/") {
		return nil, nil, ErrMegaPath
	}

	var root *mega.Node
	var err error

	switch {
	case args[0] == "mega":
		root = fs.GetRoot()
	case args[0] == "trash":
		root = fs.GetTrash()
	default:
		return nil, nil, ErrMegaPath
	}

	pathsplit := strings.Split(args[1], "/")[1:]
	l := len(pathsplit)

	if l > 0 && pathsplit[l-1] == "" {
		pathsplit = pathsplit[:l-1]
		l -= 1
	}

	if l > 0 && pathsplit[l-1] == "" {
		switch {
		case l == 1:
			pathsplit = []string{}
		default:
			pathsplit = pathsplit[:l-2]
		}
	}

	return root, &pathsplit, err
}

func (cfg *Config) Parse(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, cfg)

	if err != nil {
		return ErrConfig
	}

	return nil
}

func NewMegaClient(conf *Config) (*MegaClient, error) {
	log.SetFlags(0)
	var err error
	c := &MegaClient{
		cfg:  conf,
		mega: mega.New(),
	}

	if conf.BaseUrl != "" {
		c.mega.SetAPIUrl(conf.BaseUrl)
	}

	if conf.Retries != 0 {
		c.mega.SetRetries(conf.Retries)
	}

	if conf.DownloadWorkers != 0 {
		err = c.mega.SetDownloadWorkers(conf.DownloadWorkers)

		if err == mega.EWORKER_LIMIT_EXCEEDED {
			err = fmt.Errorf("%s : %d <= %d", err, conf.DownloadWorkers, mega.MAX_DOWNLOAD_WORKERS)
		}
	}

	if conf.UploadWorkers != 0 {
		err = c.mega.SetUploadWorkers(conf.UploadWorkers)
		if err == mega.EWORKER_LIMIT_EXCEEDED {
			err = fmt.Errorf("%s : %d <= %d", err, conf.DownloadWorkers, mega.MAX_UPLOAD_WORKERS)
		}
	}

	if conf.TimeOut != 0 {
		c.mega.SetTimeOut(time.Duration(conf.TimeOut) * time.Second)
	}

	return c, err
}
func (mc *MegaClient) Login() error {
	err := mc.mega.Login(mc.cfg.User, mc.cfg.Password)
	return err
}
