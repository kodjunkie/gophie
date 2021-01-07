package downloader

import (
	"encoding/json"
	"os"
	"path"

	"github.com/go-phie/gophie/engine"
	"github.com/iawia002/annie/downloader"
	"github.com/iawia002/annie/extractors/types"
	"github.com/iawia002/annie/request"
	"github.com/iawia002/annie/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Extract is the main function for extracting data before passing to Annie
func Extract(url, source string) ([]*types.Data, error) {

	filename, ext, err := utils.GetNameAndExt(url)
	if err != nil {
		return nil, err
	}
	size, err := request.Size(url, url)
	if err != nil {
		return nil, err
	}
	streams := map[string]*types.Stream{
		"default": {
			Parts: []*types.Part{
				{
					URL:  url,
					Size: size,
					Ext:  ext,
				},
			},
			Size: size,
		},
	}
	contentType, err := request.ContentType(url, url)
	if err != nil {
		return nil, err
	}

	return []*types.Data{
		{
			Site:    source,
			Title:   filename,
			Type:    types.DataType(contentType),
			Streams: streams,
			URL:     url,
		},
	}, nil
}

// Downloader : pausable downloader
type Downloader struct {
	URL       string // URL Source
	Dir       string // Directory to store the file
	Name      string // Name of file
	Source    string // Name of the Source
	Size      int64  // Size of the file
	Completed bool   // Status of Download
}

//TODO:  Check if Download is completed and ask for redownload confirmation

// DownloadFile : Download Files using Annie Downloader
func (f *Downloader) DownloadFile() error {
	var (
		err  error
		data []*types.Data
	)

	// Extract data to be downloaded with the streams
	data, err = Extract(f.URL, f.Source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(f.Dir, os.ModePerm)
	if err != nil {
		return err
	}

	if f.Size == 0 {
		f.Size, err = request.Size(f.URL, f.URL)
		if err != nil {
			return err
		}
	}

	for _, item := range data {
		if item.Err != nil {
			// if this error occurs, the preparation step is normal, but the data extraction is wrong.
			// the data is an empty struct.
			return item.Err
		}
		movieDownloader := downloader.New(downloader.Options{
			OutputPath: f.Dir,
			Stream:     "default",
		})
		err = movieDownloader.Download(item)
		if err != nil {
			return err
		}
	}
	log.Infof("Downloaded %s to %s", f.Name, f.Dir)
	return nil
}

// DownloadMovie : Download the movie
func DownloadMovie(movie *engine.Movie, outputDir string) error {
	url := movie.DownloadLink.String()
	downloadHandler := &Downloader{
		URL:    url,
		Name:   movie.Title,
		Source: movie.Source,
	}

	downloadHandler.Dir = path.Join(outputDir, downloadHandler.Name)
	downloadListFile := path.Join(viper.GetString("gophie_cache"), "downloadList.json")

	var (
		downloads     []Downloader
		downloadsFile *os.File
		err           error
		size          int64
	)
	//get file size
	size, err = request.Size(url, url)
	if err != nil {
		return err
	}
	downloadHandler.Size = size

	// Load Downloads if file exists
	if _, err = os.Stat(downloadListFile); err == nil {
		downloadsFile, err = os.OpenFile(downloadListFile, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		dec := json.NewDecoder(downloadsFile)
		if err = dec.Decode(&downloads); err != nil {
			return err
		}
		downloadsFile.Close()
	} else if os.IsNotExist(err) {
		// Create Download List File if it does not exists
		_, err = os.Create(downloadListFile)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// Load Config
	// Check for existing downloads
	exist := func() bool {
		for _, downloader := range downloads {
			if downloader.URL == downloadHandler.URL {
				return true
			}
		}
		return false
	}()
	// if file is not in Downloads cache, add and start the download
	// in the download cache, append it and save
	if !exist {
		log.Debug("Movie getting added to Download list")
		downloads = append(downloads, *downloadHandler)
		downloadsFile, err = os.OpenFile(downloadListFile, os.O_WRONLY, os.ModePerm)
		enc := json.NewEncoder(downloadsFile)
		if err = enc.Encode(downloads); err != nil {
			return err
		}
	}
	downloadsFile.Close()

	return downloadHandler.DownloadFile()
}
