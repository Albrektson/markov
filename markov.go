// markov
package main

import (
	"fmt"
	"bufio"
	"os"
	"math/rand"
	"bytes"
	"strings"
	"path/filepath"
	
)

//word tokens for dictionary
type token struct {
	chain map[string]int	//map of words following this token
	farChain map[string]int	//map of words following farther ahead
	chainLength int			//sum of weights in chain
	farChainLength int		//sum of weights in farChain
	count int				//how many time this word appears in corpus
}

var dictionary map[string]token
var startWords map[string]int

func main() {
	dictionary = make(map[string]token)
	startWords = make(map[string]int)
	var file *os.File
	args := os.Args[1:]	
	
	file, ok := open(args)
	if !ok { return }
	defer file.Close()
	
	read(file)
	output := generate(3) //number of sentences to generate
	fmt.Println(output)
	
	//fmt.Print("Press 'Enter' to continue...")
	//bufio.NewReader(os.Stdin).ReadBytes('\n') 
}

func open(args []string) (*os.File, bool) {
	var target string
	if len(args) > 1 {
		fmt.Println("Program only takes 1 argument.")
		return nil, false
	} else if len(args) < 1 {
		fmt.Println("No arguments given, taking first text file found.\n")
		files, _ := filepath.Glob("*.txt")
		target = "./" + files[0]
	} else {
		if strings.ContainsAny(args[0], "/") {
			target = args[0]
		} else {
			target = "./" + args[0]
		}
	}
	
	file, err := os.Open(target)
	if err != nil {
        panic(err)
    }
	return file, true
}

func read(file *os.File) {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	
	var prevWord string	//1 step back
	var pprevWord string //2-steps back
	for scanner.Scan() {
		word, good, endWord := preprocess(scanner.Text())
		if !good {continue} //skip word if it's malformed
		
		_, ok := dictionary[word]
		if !ok { 	//if word not in dictionary
			chain := make(map[string]int)
			farChain := make(map[string]int)
			t := token{chain,farChain,0,0,1} //tokenize word
			dictionary[word] = t
		}  else {	//if word found in dictionary
			t := dictionary[word]
			t.count++	//we don't really use this value
			dictionary[word] = t
		}
		
		//update chain for previous word
		if prevWord != "" {
			t := dictionary[prevWord]
			chain := t.chain
			
			//check if the word already exists before adding to map
			_, ok := chain[word]
			if !ok {
				chain[word] = 1
			} else {
				i := chain[word]
				i++
				chain[word] = i
			}
			t.chainLength++
			t.chain = chain
			dictionary[prevWord] = t
		} else {	//or add to startWords as appropriate
			_, ok := startWords[word]
			if !ok {
				startWords[word] = 1
			} else {
				i := startWords[word]
				i++
				startWords[word] = i
			}
		}
		
		if pprevWord != "" {
			t := dictionary[pprevWord]
			farChain := t.farChain
						
			_, ok := farChain[word]
			if !ok {
				farChain[word] = 1
			} else {
				i := farChain[word]
				i++
				farChain[word] = i
			}
			t.farChain = farChain
			dictionary[pprevWord] = t
		}
		
		
		
		//if prevWord != nil, we say look up that 'prevword'
		//and say 'Hey, 'word' follows after you.
		if endWord {
			prevWord = ""
			pprevWord = ""
		} else {
			pprevWord = prevWord
			prevWord = word
		}
	}
}

func preprocess(input string) (string, bool, bool) {
	ok := true
	end := false
	output := input
	var buffer bytes.Buffer
	
	//skip words with numerals
	nums := "1234567890"
	if strings.ContainsAny(input, nums) {
		ok = false
		return output, ok, end
	}
	
	//remove parenthesis
	i := strings.IndexAny(input, "()")
	if i > -1 {
		buffer.WriteString(input[:i])
		buffer.WriteString(input[i+1:])
		output = buffer.String()
	}
	
	if strings.ContainsAny(input, ".!?") {
		end = true
	}
	
	return output, ok, end
}

//generate a string that is 'length' sentences and return it	
func generate(length int) (string) {
	var buffer bytes.Buffer
	var prev1 string
	var prev2 string
	newSentence := true
	fallback := false
	
	i := 0
	for i < length {		
		r := rand.Intn(len(startWords))
		for word, _ := range startWords {
			r -= startWords[word]
			if r < 1 {
				prev1 = word
				buffer.WriteString(word)
				newSentence = false
				i++
				break
			}
		}
		
		for newSentence == false {
			
			/*
			Order-2 markov generation: correlate farChain of prev2
			and the chain of prev1 to find words that appear in both.
			*/
			
			if len(prev2) < 1 || fallback {
				//order-1 generation
				token := dictionary[prev1]
				fallback = false
				
				r = rand.Intn(token.chainLength)
				for word, count := range token.chain {
					r -= count
					if r < 1 {
						prev2 = prev1
						prev1 = word
						buffer.WriteString(" ")
						buffer.WriteString(word)
						if len(dictionary[word].chain) == 0 {
							newSentence = true
						}
						break
					}
				}
			} else {
				//order-2 generation
				token1 := dictionary[prev1]
				token2 := dictionary[prev2]
				var shortlist map[string]int
				var longlist map[string]int
				convergence := make(map[string]int)
				var convergenceLength int
				
				if len(token1.chain) < len(token2.farChain) {
					shortlist = token1.chain
					longlist = token2.farChain
				} else {
					shortlist = token2.farChain
					longlist = token1.chain
				}
				
				for word, _ := range shortlist {
					//check each word in shortlist against the longlist
					//and add to convergence map if present in both, with
					//the score from both as its value
					
					_, present := longlist[word]
					if present {
						sum := shortlist[word] + longlist[word]
						convergence[word] = sum
						convergenceLength += sum
					}	
				}
				
				if len(convergence) < 1 {
					fallback = true
				} else {
					r = rand.Intn(convergenceLength)
					for  word, count := range convergence {
						r -= count
						if r < 1 {
							prev2 = prev1
							prev1 = word
							buffer.WriteString(" ")
							buffer.WriteString(word)
							if len(dictionary[word].chain) == 0 {
								newSentence = true
							}
							break
						}
					}
				}
			}
			
		}
		buffer.WriteString("\n")
	}
	
	return buffer.String()
}