package sitemap

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Index is a structure of <sitemapindex>
type Index struct {
	XMLName xml.Name `xml:"sitemapindex"`
	Sitemap []parts  `xml:"sitemap"`
}

// parts is a structure of <sitemap> in <sitemapindex>
type parts struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// Sitemap is a structure of <sitemap>
type Sitemap struct {
	Xsi            string   `xml:"xsi,attr"`
	Image          string   `xml:"image,attr"`
	SchemaLocation string   `xml:"schemaLocation,attr"`
	Xmlns          string   `xml:"xmlns,attr"`
	XMLName        xml.Name `xml:"urlset"`
	URL            []URL    `xml:"url"`
}

// URL is a structure of <url> in <sitemap>
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float32 `xml:"priority,omitempty"`
	Image      []Image `xml:"image,omitempty"`
}

// Image is a structure of <image> in <url>
type Image struct {
	Loc     string `xml:"loc,omitempty"`
	Title   string `xml:"title,omitempty"`
	Caption string `xml:"caption,omitempty"`
	GeoLoc  string `xml:"geolocation,omitempty"`
	License string `xml:"license,omitempty"`
}

var (
	// fetch is page acquisition function
	fetch = func(URL string, options interface{}) ([]byte, error) {
		var body []byte

		res, err := http.Get(URL)
		if err != nil {
			return body, err
		}
		defer res.Body.Close()

		return io.ReadAll(res.Body)
	}

	// Time interval to be used in Index.get
	interval = time.Second
)

/*
Get is fetch and parse sitemap.xml/sitemapindex.xml

If sitemap.xml or sitemapindex.xml has some problems, This function return error.

・When sitemap.xml/sitemapindex.xml could not retrieved.
・When sitemap.xml/sitemapindex.xml is empty.
・When sitemap.xml/sitemapindex.xml has format problems.
・When sitemapindex.xml contains a sitemap.xml URL that cannot be retrieved.
・When sitemapindex.xml contains a sitemap.xml that is empty
・When sitemapindex.xml contains a sitemap.xml that has format problems.

If you want to ignore these errors, use the ForceGet function.
*/
func Get(URL string, options interface{}) (Sitemap, error) {
	data, err := fetch(URL, options)
	if err != nil {
		return Sitemap{}, err
	}

	idx, idxErr := ParseIndex(data)
	smap, smapErr := Parse(data)

	if idxErr != nil && smapErr != nil {
		if idxErr != nil {
			err = idxErr
		} else {
			err = smapErr
		}
		return Sitemap{}, fmt.Errorf("URL is not a sitemap or sitemapindex: %v", err)
	} else if idxErr != nil {
		return smap, nil
	}

	smap, err = idx.get(options, false)
	if err != nil {
		return Sitemap{}, err
	}

	return smap, nil
}

/*
ForceGet is fetch and parse sitemap.xml/sitemapindex.xml.
The difference with the Get function is that it ignores some errors.

Errors to Ignore:

・When sitemapindex.xml contains a sitemap.xml URL that cannot be retrieved.
・When sitemapindex.xml contains a sitemap.xml that is empty
・When sitemapindex.xml contains a sitemap.xml that has format problems.

Errors not to Ignore:

・When sitemap.xml/sitemapindex.xml could not retrieved.
・When sitemap.xml/sitemapindex.xml is empty.
・When sitemap.xml/sitemapindex.xml has format problems.

If you want **not** to ignore some errors, use the Get function.
*/
func ForceGet(URL string, options interface{}) (Sitemap, error) {
	data, err := fetch(URL, options)
	if err != nil {
		return Sitemap{}, err
	}

	idx, idxErr := ParseIndex(data)
	smap, smapErr := Parse(data)

	if idxErr != nil && smapErr != nil {
		if idxErr != nil {
			err = idxErr
		} else {
			err = smapErr
		}
		return Sitemap{}, fmt.Errorf("URL is not a sitemap or sitemapindex: %v", err)
	} else if idxErr != nil {
		return smap, nil
	}

	smap, err = idx.get(options, true)
	if err != nil {
		return Sitemap{}, err
	}

	return smap, nil
}

// Get Sitemap data from sitemapindex file
func (idx *Index) get(options interface{}, ignoreErr bool) (Sitemap, error) {
	var smap Sitemap

	for _, s := range idx.Sitemap {
		time.Sleep(interval)
		data, err := fetch(s.Loc, options)
		if !ignoreErr && err != nil {
			return smap, fmt.Errorf("failed to retrieve %s in sitemapindex.xml: %v", s.Loc, err)
		}

		err = xml.Unmarshal(data, &smap)
		if !ignoreErr && err != nil {
			return smap, fmt.Errorf("failed to parse %s in sitemapindex.xml: %v", s.Loc, err)
		}
	}

	return smap, nil
}

// Parse create Sitemap data from text
func Parse(data []byte) (Sitemap, error) {
	var smap Sitemap
	if len(data) == 0 {
		return smap, fmt.Errorf("sitemap.xml is empty")
	}

	err := xml.Unmarshal(data, &smap)
	return smap, err
}

// ParseIndex create Index data from text
func ParseIndex(data []byte) (Index, error) {
	var idx Index
	if len(data) == 0 {
		return idx, fmt.Errorf("sitemapindex.xml is empty")
	}

	err := xml.Unmarshal(data, &idx)
	return idx, err
}

// SetInterval change Time interval to be used in Index.get
func SetInterval(time time.Duration) {
	interval = time
}

// SetFetch change fetch closure
func SetFetch(f func(URL string, options interface{}) ([]byte, error)) {
	fetch = f
}

// Print shows the sitemap from Sitemap struct
func (smap *Sitemap) Print() ([]byte, error) {
	return xml.MarshalIndent(smap, "", "  ")
}

// Save creates the sitemap from Sitemap struct and save it to file
func (smap *Sitemap) Save(dir, file string) error {
	data, err := smap.Print()
	if err != nil {
		return err
	}

	// Add the xml header
	data = append([]byte(xml.Header), data...)

	// Create directory if it does not exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}

	_, err = os.Create(fmt.Sprintf("%s/%s", dir, file))
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s", dir, file), data, 0644)
	if err != nil {
		return err
	}

	return nil
}
