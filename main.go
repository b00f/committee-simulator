package main

import (
	"committee_simulator/sandbox"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/pactus-project/pactus/committee"
	"github.com/pactus-project/pactus/crypto"
	"github.com/pactus-project/pactus/crypto/bls"
	"github.com/pactus-project/pactus/execution"
	"github.com/pactus-project/pactus/sortition"
	"github.com/pactus-project/pactus/types/amount"
	"github.com/pactus-project/pactus/types/param"
	"github.com/pactus-project/pactus/types/tx"
	"github.com/pactus-project/pactus/types/validator"
	"github.com/pactus-project/pactus/util"
	"github.com/pactus-project/pactus/util/testsuite"
	"github.com/pelletier/go-toml"
)

type Config struct {
	CommitteeSize     int      `toml:"committee_size"`
	NumberOfDays      int      `toml:"number_of_days"`
	OfflinePercentage int      `toml:"offline_percentage"`
	Stakes            [][2]int `toml:"stakes"`
}

type ValKey struct {
	signer    *bls.ValidatorKey
	val       *validator.Validator
	sortition int
	reward    int
}

type JoinedSortitions struct {
	Trx *tx.Tx
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
	totalStake := amount.Amount(0)
	num := int32(0)
	ts := testsuite.NewTestSuiteForSeed(time.Now().Unix())
	for _, stake := range config.Stakes {
		for i := 0; i < stake[1]; i++ {
			pub, prv := ts.RandBLSKeyPair()
			val := validator.NewValidator(pub, num)
			val.AddToStake(amount.Amount(stake[0] * 10e9))
			val.UpdateLastBondingHeight(uint32(i))
			val.UpdateLastSortitionHeight(uint32(i))

			valKeys = append(valKeys, ValKey{
				signer:    bls.NewValidatorKey(prv),
				val:       val,
				sortition: 0,
			})

			num++
			totalStake += val.Stake()

			mockValidators[pub.ValidatorAddress()] = val
		}
	}

	totalStake += amount.Amount(float64(totalStake) * float64(config.OfflinePercentage) / 100)

	vals := []*validator.Validator{}
	for i := 0; i < config.CommitteeSize; i++ {
		valKeys[i].sortition++
		vals = append(vals, valKeys[i].val)
	}
	cmt, err := committee.NewCommittee(vals, config.CommitteeSize, vals[0].Address())
	if err != nil {
		panic(err)
	}

	evaluateSortition := func(height uint32, cmt committee.Committee,
		seed sortition.VerifiableSeed, start, len int) []*tx.Tx {
		sortitions := []*tx.Tx{}
		for i := start; i < start+len; i++ {
			valKey := valKeys[i]

			rnd := util.RandInt64(totalStake.ToNanoPAC())
			if rnd < valKey.val.Power() {
				valKeys[i].sortition++
				trx := tx.NewSortitionTx(height, valKey.val.Address(), ts.RandProof())
				// sb := trx.SignBytes()
				// sig := valKey.signer.Sign(sb)
				// trx.SetSignature(sig)

				sortitions = append(sortitions, trx)
			}

			// Our sortition algorithm is too slow for this simulator.
			// We can replace it with a safe random number generator.
			// You can uncomment this code, but it gets too slow.
			//
			// ok, _ := sortition.EvaluateSortition(seed, valKey.signer, totalStake, valKey.val.Power())
			// if ok {
			// 	valKeys[i].sortition++
			// 	trx := tx.NewSortitionTx(stamp, valKey.val.Sequence()+1, valKey.val.Address(), ts.RandProof())
			// 	valKey.signer.SignMsg(trx)
			//
			// sortitions = append(sortitions, trx)
			// }
		}
		return sortitions
	}

	seed := ts.RandSeed()
	startHeight := uint32(10000)
	sb := &sandbox.MockSandbox{
		MockCommittee:    cmt,
		MockValidators:   mockValidators,
		MockParams:       *params,
		CurHeight:        startHeight,
		JoinedValidators: []*validator.Validator{},
	}
	for height := startHeight; height < uint32(config.NumberOfDays*8640)+startHeight; height++ {
		sb.CurHeight = height

		if height%1000 == 0 {
			fmt.Printf("Height %v\n", height)
		}

		threads := []*Thread{}
		threadCount := 8
		for i := 0; i < threadCount; i++ {
			threads = append(threads, NewThread(evaluateSortition))
		}
		totalValNum := len(valKeys)
		length := totalValNum / threadCount
		for i, th := range threads {
			if i == threadCount-1 {
				newLen := length + totalValNum%threadCount
				th.Start(height, cmt, seed, length*i, newLen)
			} else {
				th.Start(height, cmt, seed, length*i, length)
			}
		}
		allSortitions := []*tx.Tx{}
		for _, th := range threads {
			allSortitions = append(allSortitions, th.Join()...)
		}

		for _, trx := range allSortitions {
			err := execution.Execute(trx, sb)
			if err != nil {
				val := sb.Validator(trx.Payload().Signer())
				fmt.Fprintf(logFile, "height: %v, val %v, err: %v\n", height, val.Number(), err.Error())
			}
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
			fmt.Fprintf(logFile, "]")
		}

		cmt.Update(0, sb.JoinedValidators)
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
		fmt.Fprintf(csvFile, "%v, %v, %v, %v\n", i+1, valPrv.val.Stake().ToNanoPAC(), valPrv.reward, valPrv.sortition)

		totalRewards += valPrv.reward
		totalSortitions += valPrv.sortition
	}

	fmt.Printf("Total Rewards: %v\n", totalRewards)
	fmt.Printf("Total Sortitions: %v\n", totalSortitions)
	fmt.Printf("Expected Sortitions: %v\n", config.NumberOfDays*8640)

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
