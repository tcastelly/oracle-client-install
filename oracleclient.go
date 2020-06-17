package oracleclient

import (
  "archive/zip"
  "errors"
  "fmt"
  "github.com/dustin/go-humanize"
  "github.com/tcastelly/oracle-client-install/config"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path"
  "path/filepath"
  "regexp"
  "strings"
  "sync"
)

type writeCounter struct {
  Total uint64

  Info string;
}

func (wc *writeCounter) Write(p []byte) (int, error) {
  n := len(p)
  wc.Total += uint64(n)
  wc.PrintProgress()
  return n, nil
}

func (wc writeCounter) PrintProgress() {
  // Clear the line by using a character return to go back to the start and remove
  // the remaining characters by filling it with spaces
  fmt.Printf("\r%s", strings.Repeat(" ", 35))

  // Return again and print current status of download
  // We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
  fmt.Printf("\r%s %s completed", wc.Info, humanize.Bytes(wc.Total))
}

func downloadFile(filename string, url string, info string) error {
  // Create the file, but give it a tmp file extension, this means we won't overwrite a
  // file until it's downloaded, but we'll remove the tmp extension once downloaded.
  out, err := os.Create(filename + ".tmp")
  if err != nil {
    return err
  }

  // Get the data
  resp, err := http.Get(url)
  if err != nil {
    out.Close()
    return err
  }
  defer resp.Body.Close()

  // Create our progress reporter and pass it to be used alongside our writer
  counter := &writeCounter{
    0,

    info,
  }
  if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
    out.Close()
    return err
  }

  // The progress use the same line so print a new line once it's finished downloading
  fmt.Print("\n")

  // close the file without defer so it can happen before rename()
  out.Close()

  if err = os.Rename(filename+".tmp", filename); err != nil {
    return err
  }
  return nil
}

func unzip(src string, dest string) ([]string, error) {
  var filenames []string

  r, err := zip.OpenReader(src)
  if err != nil {
    return filenames, err
  }
  defer r.Close()

  dest = path.Dir(dest)

  for _, f := range r.File {
    // Store filename/path for returning and using later on
    fpath := filepath.Join(dest, f.Name)

    // Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
    if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
      return filenames, fmt.Errorf("%s: illegal file path", fpath)
    }

    filenames = append(filenames, fpath)

    if f.FileInfo().IsDir() {
      // Make Folder
      os.MkdirAll(fpath, os.ModePerm)
      continue
    }

    // Make File
    if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
      return filenames, err
    }

    outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
    if err != nil {
      return filenames, err
    }

    rc, err := f.Open()
    if err != nil {
      return filenames, err
    }

    _, err = io.Copy(outFile, rc)

    // Close the file without defer to close before next iteration of loop
    outFile.Close()
    rc.Close()

    if err != nil {
      return filenames, err
    }
  }
  return filenames, nil
}

func findInstanclientPath(rootPath string) (string, error) {
  folders, err := ioutil.ReadDir(rootPath)
  if err != nil {
    return "", err
  }

  var validInstantclientPath = regexp.MustCompile(`^instantclient_[\S]+$`)

  found := false
  i := 0
  for !found && i < len(folders) {
    f := folders[i]
    found = f.IsDir() && validInstantclientPath.MatchString(f.Name())

    i += 1
  }

  if found {
    return folders[i-1].Name(), nil
  }

  return "", fmt.Errorf("instantclient not found")
}

func rename(rootPath string) error {
  instanclientPath, err := findInstanclientPath(rootPath)
  if err != nil {
    return err
  }

  os.Rename(path.Join(rootPath, instanclientPath), path.Join(rootPath, "instantclient"))

  return nil
}

func clean(toRemove []string) {
  for _, f := range toRemove {
    os.Remove(f)
  }
}

func Uninstall(outputDir string) error {
  matched, err := regexp.MatchString("\\.$|\\./$+", outputDir)
  if err != nil {
    return err
  }

  if matched {
    return errors.New("Current dirrectory is not allowed")
  }

  fmt.Println("clean previous install")
  os.RemoveAll(outputDir)

  return nil
}

func Install(outputDir string) error {
  configFiles, err := config.NewLinuxConfig()
  if err != nil {
    return err
  }

  var wgDownload sync.WaitGroup
  wgDownload.Add(2)

  basiclite := filepath.Base(configFiles.InstantclientBasic)
  go func() {
    downloadFile(basiclite, configFiles.InstantclientBasic, "Download basic:")
    wgDownload.Done()
  }()

  sdk := filepath.Base(configFiles.InstantclientSdk)
  go func() {
    downloadFile(sdk, configFiles.InstantclientSdk, "Download sdk:")
    wgDownload.Done()
  }()

  wgDownload.Wait()

  var wgInstall sync.WaitGroup
  wgInstall.Add(2)
  fmt.Println("Installing ...")
  go func() {
    unzip(basiclite, path.Join(outputDir, basiclite))
    wgInstall.Done()
  }()

  go func() {
    unzip(sdk, path.Join(outputDir, sdk))
    wgInstall.Done()
  }()

  wgInstall.Wait()

  err = rename(outputDir)
  if err != nil {
    return err
  }

  clean([]string{
    basiclite,
    sdk,
  })

  fmt.Printf("Driver installed: %s", outputDir)

  return nil
}
