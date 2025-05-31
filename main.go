package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/clausecker/nfc/v2"
)

type Reader struct {
	name   string
	device string
	active map[string]bool
	mu     sync.Mutex
}

func (r *Reader) monitor(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		dev, err := nfc.Open(r.device)
		if err != nil {
			log.Printf("[%s] Failed to open device %s: %v", r.name, r.device, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("[%s] Device opened successfully", r.name)

		if err := dev.InitiatorInit(); err != nil {
			log.Printf("[%s] Failed to initialize as initiator: %v", r.name, err)
			dev.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("[%s] Device initialized as initiator", r.name)
		
		r.pollTargets(dev)
		dev.Close()
		time.Sleep(1 * time.Second)
	}
}

func (r *Reader) pollTargets(dev nfc.Device) {
	modulations := []nfc.Modulation{
		{Type: nfc.ISO14443a, BaudRate: nfc.Nbr106},
		{Type: nfc.ISO14443b, BaudRate: nfc.Nbr106},
		{Type: nfc.Felica, BaudRate: nfc.Nbr212},
		{Type: nfc.Felica, BaudRate: nfc.Nbr424},
		{Type: nfc.Jewel, BaudRate: nfc.Nbr106},
	}

	log.Printf("[%s] Starting polling loop", r.name)

	for {
		// Track currently seen tags
		currentTags := make(map[string]bool)

		// Try each modulation type
		for _, mod := range modulations {
			// Use recover to handle any panics from the library
			func() {
				defer func() {
					if err := recover(); err != nil {
						// Silently ignore panics - the pn71xx driver might return incompatible data
					}
				}()

				// Try listing passive targets
				targets, err := dev.InitiatorListPassiveTargets(mod)
				if err != nil {
					return
				}

				// Process found targets
				for _, target := range targets {
					uid := r.getUID(target)
					if uid != "" {
						currentTags[uid] = true
						
						r.mu.Lock()
						if !r.active[uid] {
							r.active[uid] = true
							r.mu.Unlock()
							
							fmt.Printf("[%s] TAG ARRIVED - UID: %s, Type: %s\n", 
								r.name, uid, r.getTargetType(target))
							r.printTargetInfo(target)
						} else {
							r.mu.Unlock()
						}
					}
				}
			}()
		}

		// Check for departed tags
		r.mu.Lock()
		for uid := range r.active {
			if !currentTags[uid] {
				delete(r.active, uid)
				fmt.Printf("[%s] TAG DEPARTED - UID: %s\n", r.name, uid)
			}
		}
		r.mu.Unlock()

		// Small delay between polling cycles
		time.Sleep(200 * time.Millisecond)
	}
}

func (r *Reader) getUID(target nfc.Target) string {
	defer func() {
		if err := recover(); err != nil {
			// Return empty string if we can't get UID
		}
	}()

	switch t := target.(type) {
	case *nfc.ISO14443aTarget:
		return fmt.Sprintf("%X", t.UID)
	case *nfc.ISO14443bTarget:
		return fmt.Sprintf("%X", t.ApplicationData)
	case *nfc.FelicaTarget:
		return fmt.Sprintf("%X", t.ID)
	case *nfc.JewelTarget:
		return fmt.Sprintf("%X", t.ID)
	case *nfc.ISO14443biClassTarget:
		return fmt.Sprintf("%X", t.UID)
	default:
		return ""
	}
}

func (r *Reader) getTargetType(target nfc.Target) string {
	switch target.(type) {
	case *nfc.ISO14443aTarget:
		return "ISO14443A"
	case *nfc.ISO14443bTarget:
		return "ISO14443B"
	case *nfc.FelicaTarget:
		return "FeliCa"
	case *nfc.JewelTarget:
		return "Jewel"
	case *nfc.ISO14443biClassTarget:
		return "ISO14443B iClass"
	default:
		return "Unknown"
	}
}

func (r *Reader) printTargetInfo(target nfc.Target) {
	defer func() {
		if err := recover(); err != nil {
			// Ignore errors in printing
		}
	}()

	switch t := target.(type) {
	case *nfc.ISO14443aTarget:
		fmt.Printf("  ATQA: %04X\n", t.Atqa)
		fmt.Printf("  SAK: %02X\n", t.Sak)
		if len(t.Ats) > 0 {
			fmt.Printf("  ATS: %X\n", t.Ats)
		}
	case *nfc.ISO14443bTarget:
		fmt.Printf("  PUPI: %X\n", t.Pupi)
		fmt.Printf("  Application Data: %X\n", t.ApplicationData)
		fmt.Printf("  Protocol Info: %X\n", t.ProtocolInfo)
	case *nfc.FelicaTarget:
		fmt.Printf("  ID: %X\n", t.ID)
		fmt.Printf("  Pad: %X\n", t.Pad)
		fmt.Printf("  System Code: %X\n", t.SysCode)
	}
}

func main() {
	fmt.Println("NFC Monitor starting...")

	// Use pn71xx driver for the PN7150 chips
	reader1 := &Reader{
		name:   "Reader 1", 
		device: "pn71xx:/dev/pn5xx_i2c0",
		active: make(map[string]bool),
	}

	reader2 := &Reader{
		name:   "Reader 2",
		device: "pn71xx:/dev/pn5xx_i2c1", 
		active: make(map[string]bool),
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go reader1.monitor(&wg)
	go reader2.monitor(&wg)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nShutting down...")
	os.Exit(0)
}