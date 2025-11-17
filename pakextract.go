package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PakHeader struct {
	Magic     [4]byte
	DirOffset uint32
	DirLength uint32
}

type PakDirectory struct {
	Name   [56]byte
	Offset uint32
	Length uint32
}

type PakFileEntry struct {
	Name   string
	Offset uint32
	Length uint32
}

type Flags struct {
	OutputDir string
	Verbose   bool
}

var flags Flags

func CollectPakFileEntries(pakFile *os.File) ([]PakFileEntry, error) {
	var header PakHeader
	err := binary.Read(pakFile, binary.LittleEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("Failed to read PAK header: %w", err)
	}

	if string(header.Magic[:]) != "PACK" {
		return nil, fmt.Errorf("Invalid file magic: %s", header.Magic)
	}

	_, err = pakFile.Seek(int64(header.DirOffset), 0)
	if err != nil {
		return nil, fmt.Errorf("Failed to seek to directory offset: %w", err)
	}

	numEntries := header.DirLength / 64
	entries := make([]PakFileEntry, 0, numEntries)

	for i := range numEntries {
		var dirEntry PakDirectory
		err = binary.Read(pakFile, binary.LittleEndian, &dirEntry)
		if err != nil {
			return nil, fmt.Errorf("Failed to read directory entry %d: %w", i, err)
		}

		name := string(dirEntry.Name[:])
		name = strings.TrimRight(name, "\x00")

		entries = append(entries, PakFileEntry{
			Name:   name,
			Offset: dirEntry.Offset,
			Length: dirEntry.Length,
		})
	}

	return entries, nil
}

func ExtractPakFileEntry(file *os.File, entry PakFileEntry) error {
	_, err := file.Seek(int64(entry.Offset), 0)
	if err != nil {
		return fmt.Errorf("Failed to seek file data: %w", err)
	}

	data := make([]byte, entry.Length)
	_, err = file.Read(data)
	if err != nil {
		return fmt.Errorf("Failed to read file data: %w", err)
	}

	extractPath := filepath.Join(flags.OutputDir, entry.Name)

	parts := strings.Split(extractPath, "/")
	dirPath := strings.Join(parts[:len(parts)-1], "/")

	if dirPath != "" {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return fmt.Errorf("Failed to create directories: %w", err)
		}
	}

	if flags.Verbose {
		fmt.Printf("Extracting %s\n", entry.Name)
	}

	return os.WriteFile(extractPath, data, 0644)
}

func main() {
	flag.StringVar(&flags.OutputDir, "output", "", "directory to extract files to")
	flag.BoolVar(&flags.Verbose, "verbose", false, "enable verbose output")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: pakextract [--output=<directory>] [--verbose] <pak_file>")
		return
	}

	pakFile, err := os.Open(flag.Args()[0])
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return
	}
	defer pakFile.Close()

	entries, err := CollectPakFileEntries(pakFile)
	if err != nil {
		fmt.Printf("Failed to read directory entries: %v\n", err)
		return
	}

	if flags.Verbose {
		fmt.Printf("Found %d files\n", len(entries))
	}

	for _, entry := range entries {
		ExtractPakFileEntry(pakFile, entry)
	}
}
