package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Instances
type Instance struct {
	ID          int
	Status      string
	PartyCount  int
	TotalTime   int64
	CurrentTime int64
	mutex       sync.Mutex
}

// Player Queue
type Queue struct {
	Tanks   int
	Healers int
	DPS     int
	mutex   sync.Mutex
}

// Config from user input
type Config struct {
	NumInstances  int
	MinClearTime  int
	MaxClearTime  int
	InitialTanks  int
	InitialHealers int
	InitialDPS    int
}

func main() {
	var config Config

	// Get user input
	fmt.Print("Enter maximum number of concurrent instances (n): ")
	fmt.Scan(&config.NumInstances)

	fmt.Print("Enter number of tank players in the queue (t): ")
	fmt.Scan(&config.InitialTanks)

	fmt.Print("Enter number of healer players in the queue (h): ")
	fmt.Scan(&config.InitialHealers)

	fmt.Print("Enter number of DPS players in the queue (d): ")
	fmt.Scan(&config.InitialDPS)

	fmt.Print("Enter minimum time before an instance is finished (t1): ")
	fmt.Scan(&config.MinClearTime)

	fmt.Print("Enter maximum time before an instance is finished (t2): ")
	fmt.Scan(&config.MaxClearTime)

	// Initialize instances
	instances := make([]*Instance, config.NumInstances)
	for i := 0; i < config.NumInstances; i++ {
		instances[i] = &Instance{
			ID:         i + 1,
			Status:     "empty",
			PartyCount: 0,
			TotalTime:  0,
		}
	}

	// Initialize player queue
	queue := &Queue{
		Tanks:   config.InitialTanks,
		Healers: config.InitialHealers,
		DPS:     config.InitialDPS,
	}

	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	
	// Display status goroutine
	wg.Add(1)
	go displayStatus(instances, queue, &wg, stopChan)

	// Manage instances goroutine
	wg.Add(1)
	go manageInstances(instances, queue, &wg, stopChan, config)

	for {
		totalPartiesFormed := 0
		activeDungeons := 0
		for _, inst := range instances {
			inst.mutex.Lock()
			totalPartiesFormed += inst.PartyCount
			if inst.Status == "active" {
				activeDungeons++
			}
			inst.mutex.Unlock()
		}
		

		canFormParty := queue.Tanks >= 1 && queue.Healers >= 1 && queue.DPS >= 3
		
		// Termination rules = no active dungeons and no more party can be formed
		if activeDungeons == 0 && !canFormParty {
			break
		}
		
	
		time.Sleep(1 * time.Second)
	}

	close(stopChan)
	wg.Wait()

	fmt.Println("\nConcurrency Expedition Complete!")
	
	totalPartiesFormed := 0
	for _, inst := range instances {
		totalPartiesFormed += inst.PartyCount
	}
	
	totalPlayers := totalPartiesFormed * 5 
	
	fmt.Printf("Total parties formed: %d (total %d players)\n", totalPartiesFormed, totalPlayers)
	
	// Remaining players
	queue.mutex.Lock()
	fmt.Printf("Players remaining in queue: %d Tanks, %d Healers, %d DPS (Total: %d)\n", 
		queue.Tanks, queue.Healers, queue.DPS, queue.Tanks + queue.Healers + queue.DPS)
	queue.mutex.Unlock()
	
	// Instance information
	fmt.Println("\nInstance Information:")
	for _, inst := range instances {
		fmt.Printf("Instance %d: Served %d parties, Total time: %d seconds\n", 
			inst.ID, inst.PartyCount, inst.TotalTime)
	}
}

// Loop checks if parties can be formed
func manageInstances(instances []*Instance, queue *Queue, wg *sync.WaitGroup, stopChan chan struct{}, config Config) {
	defer wg.Done()
	// just check if can form party every 100 ms
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			formParties(instances, queue, config)
		}
	}
}

// Function form parties and assign to available instances
func formParties(instances []*Instance, queue *Queue, config Config) {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	
	// Loop forming parties as long as we have enough players and empty instances
	for queue.Tanks >= 1 && queue.Healers >= 1 && queue.DPS >= 3 {
		// Look for empty then mark as active to loop again
		var emptyInstance *Instance
		for _, inst := range instances {
			inst.mutex.Lock()
			if inst.Status == "empty" {
				emptyInstance = inst
				inst.Status = "active"
				inst.mutex.Unlock()
				break
			}
			inst.mutex.Unlock()
		}
		
		// If no empty instance then break
		if emptyInstance == nil {
			break
		}
		
		queue.Tanks--
		queue.Healers--
		queue.DPS -= 3

		go startInstance(emptyInstance, config)
	}
}

// Dungeon instance simulation
func startInstance(instance *Instance, config Config) {
	
	clearTime := rand.Intn(config.MaxClearTime-config.MinClearTime+1) + config.MinClearTime

	instance.mutex.Lock()
	instance.CurrentTime = int64(clearTime)
	partyTime := int64(clearTime)
	instance.mutex.Unlock()
	
	// Countdown timer, loop every 1 sec
	for i := clearTime; i > 0; i-- {
		instance.mutex.Lock()
		instance.CurrentTime = int64(i)
		instance.mutex.Unlock()
		time.Sleep(1 * time.Second)
	}
	
	// Update instance information for summary
	instance.mutex.Lock()
	instance.Status = "empty"
	instance.TotalTime += partyTime
	instance.PartyCount++
	instance.mutex.Unlock()
}

// Loop displays the status of all instances and the player queue
func displayStatus(instances []*Instance, queue *Queue, wg *sync.WaitGroup, stopChan chan struct{}) {
	defer wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			fmt.Print("\033[H\033[2J") // Clear screen mainly for golang i think

			activeCount := 0
			emptyCount := 0
			totalPartiesDone := 0
			
			fmt.Println("=== PARTY STATUS ===")
			for _, inst := range instances {
				inst.mutex.Lock()
				status := inst.Status
				timeInfo := ""
				
				if status == "active" {
					timeInfo = fmt.Sprintf(" (Time remaining: %d seconds)", inst.CurrentTime)
					activeCount++
				} else {
					emptyCount++
				}
				
				fmt.Printf("PARTY %d: %s%s\n", inst.ID, status, timeInfo)
				totalPartiesDone += inst.PartyCount
				inst.mutex.Unlock()
			}
			
			// Real-time displaying of information
			fmt.Printf("\nTotal Possible Parties: %d | Active: %d | Empty: %d | Total Parties Done: %d\n", 
				len(instances), activeCount, emptyCount, totalPartiesDone)
			
			queue.mutex.Lock()
			fmt.Println("\n=== QUEUE STATUS ===")
			fmt.Printf("Tanks: %d, Healers: %d, DPS: %d\n", queue.Tanks, queue.Healers, queue.DPS)

			canFormParty := "No"
			if queue.Tanks >= 1 && queue.Healers >= 1 && queue.DPS >= 3 {
				canFormParty = "Yes"
			}
			fmt.Printf("Can form another party: %s\n", canFormParty)
			queue.mutex.Unlock()
		}
	}
}