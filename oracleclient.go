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

func clean(toRemove []string) error {
  for _, f := range toRemove {
    return os.Remove(f)
  }

  return nil
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
  return os.RemoveAll(outputDir)
}

func Install(outputDir string) error {
  var err error

  var wg sync.WaitGroup

  // success / error
  wg.Add(2)

  successCh := make(chan string)
  errCh := make(chan error)

  go InstallWithCh(outputDir, successCh, errCh)

  go func(errCh <-chan error) {
    err = <-errCh
    wg.Done()
  }(errCh)

  go func(ch <-chan string) {
    for str := range ch {
      fmt.Println(str)
    }
    wg.Done()
  }(successCh)

  wg.Wait()

  return err
}

func InstallWithCh(outputDir string, ch chan<- string, errCh chan<- error) {
  defer close(ch)
  defer close(errCh)

  configFiles, err := config.NewLinuxConfig()
  if err != nil {
    errCh <- err
  }

  var wgDownload sync.WaitGroup

  // two downloads
  wgDownload.Add(2)

  basiclite := filepath.Base(configFiles.InstantclientBasic)
  go func() {
    err := downloadFile(basiclite, configFiles.InstantclientBasic, "Download basic:")
    if err != nil {
      errCh <- err
    }
    ch <- "Basic downloaded"
    defer wgDownload.Done()
  }()

  sdk := filepath.Base(configFiles.InstantclientSdk)
  go func() {
    err := downloadFile(sdk, configFiles.InstantclientSdk, "Download sdk:")
    if err != nil {
      errCh <- err
    }
    ch <- "SDK downloaded"
    defer wgDownload.Done()
  }()

  wgDownload.Wait()

  var wgInstall sync.WaitGroup

  // install + unzip
  wgInstall.Add(2)

  ch <- "Installing ..."
  go func() {
    _, err := unzip(basiclite, path.Join(outputDir, basiclite))
    if err != nil {
      errCh <- err
    }
    defer wgInstall.Done()
  }()

  go func() {
    _, err := unzip(sdk, path.Join(outputDir, sdk))
    if err != nil {
      errCh <- err
    }
    defer wgInstall.Done()
  }()

  wgInstall.Wait()

  err = rename(outputDir)
  if err != nil {
    errCh <- err
  }

  err = clean([]string{
    basiclite,
    sdk,
  })
  if err != nil {
    errCh <- err
  }

  doneMsg := fmt.Sprintf("Driver installed: %s\n", outputDir)
  ch <- doneMsg
}
