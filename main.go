package main

import (
	"committee_simulator/sandbox"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pactus-project/pactus/committee"
	"github.com/pactus-project/pactus/crypto"
	"github.com/pactus-project/pactus/crypto/bls"
	"github.com/pactus-project/pactus/crypto/hash"
	"github.com/pactus-project/pactus/execution"
	"github.com/pactus-project/pactus/sortition"
	"github.com/pactus-project/pactus/types/param"
	"github.com/pactus-project/pactus/types/tx"
	"github.com/pactus-project/pactus/types/validator"
	"github.com/pactus-project/pactus/util"
	"github.com/pelletier/go-toml"
)

type Config struct {
	CommitteeSize     int      `toml:"committee_size"`
	NumberOfDays      int      `toml:"number_of_days"`
	OfflinePercentage int      `toml:"offline_percentage"`
	Stakes            [][2]int `toml:"stakes"`
}

type ValKey struct {
	signer    crypto.Signer
	val       *validator.Validator
	sortition int
	reward    int
}

type JoinedValKey struct {
	ValKey    *ValKey
	BlockHash hash.Hash
}

func hasDuplicates(vector []*validator.Validator) bool {
	countMap := make(map[int32]int)

	for _, val := range vector {
		countMap[val.Number()]++
		if countMap[val.Number()] > 1 {
			return true
		}
	}

	return false
}

func main() {
	logCommittee := true
	data, err := os.ReadFile("config.toml")
	if err != nil {
		fmt.Println("Error ", err.Error())
		return
	}

	// Create the output folder if it doesn't exist
	err = os.MkdirAll("output", os.ModePerm)
	if err != nil {
		fmt.Println("Error creating output folder:", err)
		return
	}

	// Generate the file name with current date and time
	currentTime := time.Now()
	csvFileName := currentTime.Format("output/20060102_150405") + ".csv"
	logFileName := currentTime.Format("output/20060102_150405") + ".log"
	// Open the file for writing
	logFile, err := os.Create(logFileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	config := Config{}
	err = toml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Error ", err.Error())
		return
	}
	params := param.DefaultParams()
	params.CommitteeSize = config.CommitteeSize

	mockValidators := make(map[crypto.Address]*validator.Validator)
	valKeys := []ValKey{}
	totalStake := int64(0)
	num := int32(0)
	for _, stake := range config.Stakes {
		for i := 0; i < stake[1]; i++ {
			pub, prv := bls.GenerateTestKeyPair()
			val := validator.NewValidator(pub, num)
			val.AddToStake(int64(stake[0]) * 10e9)
			val.UpdateLastBondingHeight(uint32(i))
			val.UpdateLastSortitionHeight(uint32(i))

			valKeys = append(valKeys, ValKey{
				signer:    crypto.NewSigner(prv),
				val:       val,
				sortition: 0,
			})

			num++
			totalStake += val.Stake()

			mockValidators[pub.Address()] = val
		}
	}

	totalStake += int64(float64(totalStake) * float64(config.OfflinePercentage) / 100)

	vals := []*validator.Validator{}
	for i := 0; i < config.CommitteeSize; i++ {
		valKeys[i].sortition++
		vals = append(vals, valKeys[i].val)
	}
	cmt, err := committee.NewCommittee(vals, config.CommitteeSize, vals[0].Address())
	if err != nil {
		panic(err)
	}

	evaluateSortition := func(hash hash.Hash, cmt committee.Committee,
		seed sortition.VerifiableSeed, start, len int) []JoinedValKey {
		joined := []JoinedValKey{}
		for i := start; i < start+len; i++ {
			valKey := valKeys[i]

			rnd := util.RandInt64(totalStake)
			if rnd < valKey.val.Power() {
				valKeys[i].sortition++
				joined = append(joined, JoinedValKey{
					ValKey:    &valKey,
					BlockHash: hash,
				})
			}

			// Our sortition algorithm is too slow for this simulator.
			// We can replace it with a safe random number generator.
			// You can uncomment this code, but it gets too slow.
			//
			// ok, _ := sortition.EvaluateSortition(seed, valKey.signer, totalStake, valKey.val.Power())
			// if ok {
			// 	valKeys[i].sortition++
			// 	valKeys[i].val.UpdateLastJoinedHeight(height)
			// 	joined = append(joined, valKey.val)
			// }
		}
		return joined
	}

	seed := sortition.GenerateRandomSeed()
	joined := []JoinedValKey{}
	blockHashes := make([]hash.Hash, 0, uint32(config.NumberOfDays*8640)+1)
	exe := execution.NewExecutor()
	startHeight := uint32(10000)
	sb := &sandbox.MockSandbox{
		MockCommittee:    cmt,
		MockValidators:   mockValidators,
		MockParams:       params,
		BlockHashes:      blockHashes,
		StartHeight:      startHeight,
		JoinedValidators: map[crypto.Address]bool{},
	}
	for height := startHeight; height < uint32(config.NumberOfDays*8640)+startHeight; height++ {
		blockHash := hash.GenerateTestHash()
		sb.BlockHashes = append(sb.BlockHashes, blockHash)
		if height%100 == 0 {
			fmt.Printf("Height %v\n", height)
		}

		threads := []*Thread{}
		threadCount := 4
		for i := 0; i < threadCount; i++ {
			threads = append(threads, NewThread(evaluateSortition))
		}
		totalValNum := len(valKeys)
		length := totalValNum / threadCount
		for i, th := range threads {
			if i == threadCount-1 {
				newLen := length + totalValNum%threadCount
				th.Start(blockHash, cmt, seed, length*i, newLen)
			} else {
				th.Start(blockHash, cmt, seed, length*i, length)
			}
		}
		for _, th := range threads {
			joined = append(joined, th.Join()...)
		}

		canJoin := []*validator.Validator{}
		removeIndex := []int{}
		for i, j := range joined {
			trx := tx.NewSortitionTx(j.BlockHash.Stamp(), j.ValKey.val.Sequence()+1, j.ValKey.val.Address(), sortition.GenerateRandomProof())
			j.ValKey.signer.SignMsg(trx)
			err := exe.Execute(trx, sb)
			if err == nil {
				canJoin = append(canJoin, j.ValKey.val)
				j.ValKey.val.IncSequence()

				removeIndex = append(removeIndex, i)
			} else {
				if strings.Contains(err.Error(), "Invalid stamp. The transactions failed in 7 blocks due to a sudden influx of sortition evaluations.") {
					fmt.Fprintf(logFile, "height: %v, val %v, err: %v\n", height, j.ValKey.val.Number(), err.Error())
					removeIndex = append(removeIndex, i)
				} else {
					break
				}
			}
		}

		for n := len(removeIndex) - 1; n >= 0; n-- {
			joined = append(joined[:n], joined[n+1:]...)
		}

		proposer := cmt.Proposer(0).Number()
		valKeys[proposer].reward++

		if logCommittee {
			fmt.Fprintf(logFile, "%v [", height)
			for _, val := range cmt.Validators() {
				if proposer == val.Number() {
					fmt.Fprintf(logFile, "%v*-", val.Number())
				} else {
					fmt.Fprintf(logFile, "%v-", val.Number())
				}
			}
			fmt.Fprintf(logFile, "] <- (")
			for _, j := range canJoin {
				fmt.Fprintf(logFile, "%v-", j.Number())
			}
			fmt.Fprintf(logFile, ")\n")
		}

		cmt.Update(0, canJoin)
		sb.Reset()
	}

	// Open the csvFile for writing
	csvFile, err := os.Create(csvFileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	fmt.Fprintf(csvFile, "Validator,Stake,Reward,Sortition\n")
	totalRewards := 0
	totalSortitions := 0
	for i, valPrv := range valKeys {
		fmt.Fprintf(csvFile, "%v, %v, %v, %v\n", i+1, valPrv.val.Stake()/10e9, valPrv.reward, valPrv.sortition)

		totalRewards += valPrv.reward
		totalSortitions += valPrv.sortition
	}

	fmt.Printf("totalRewards: %v\n", totalRewards)
	fmt.Printf("totalSortitions: %v\n", totalSortitions)

	if totalRewards != 8640*config.NumberOfDays {
		panic("invalid data")
	}
	csvFile.Close()

	cmd := exec.Command("python3", "graph.py", csvFileName,
		strconv.Itoa(config.CommitteeSize),
		strconv.Itoa(config.OfflinePercentage),
		strconv.Itoa(config.NumberOfDays))
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}
