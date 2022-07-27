package cmd

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"
)

type Gadget struct {
	name            string
	json            string
	parametersCount int
}

const gadgetsFilename = "pylons.gadgets"

const err_duplicateName = "Duplicate gadget name: "
const err_reservedName = "Can't register a gadget of reserved name "
const err_noHeader = "pylons.gadgets file does not start with a valid gadget header"
const err_badHeader = "Not a valid gadget header: \n"

var builtinGadgets []Gadget = []Gadget{
	{
		"price",
		`"coinInputs": [
			{
				"coins" : [
					{
					"denom": "%0",
					"amount": "%1"
					}
				]
			}
		]`,
		2,
	},
	{
		"no_input",
		`"coinInputs": [],
		"itemInputs": []`,
		0,
	},
	{
		"no_coin_input",
		`"coinInputs": []`,
		0,
	},
	{
		"no_item_input",
		`"itemInputs": []`,
		0,
	},
	{
		"id_name",
		`"id": "%0",
		"name": "%1",`,
		2,
	},
	{
		"no_coin_output",
		`"coinOutputs": []`,
		0,
	},
	{
		"no_item_output",
		`"itemOutputs": []`,
		0,
	},
	{
		"no_item_modify_output",
		`"itemModifyOutputs": []`,
		0,
	},
	{
		"no_coin_or_item_output",
		`"coinOutputs": [],
		"itemOutputs": []`,
		0,
	},
	{
		"no_coin_or_item_modify_output",
		`"coinOutputs": [],
		"itemModifyOutputs": []`,
		0,
	},
	{
		"no_item_or_item_modify_output",
		`"itemOutputs": [],
		"itemModifyOutputs": []`,
		0,
	},
	{
		"solo_output",
		`"outputs": [
			{
				"entryIds": [
					"%0"
				],
				"weight": 1
			}
		],`,
		1,
	},
	{
		"no_output",
		`"entries": {},
		"outputs": [],`,
		0,
	},
}

var reservedNames = []string{"include"}

// one iteration
func loadGadgetsForPath(p string, gadgets *[]Gadget) (string, string, *[]Gadget) {
	fpath := path.Join(p, gadgetsFilename)
	_, err := os.Stat(fpath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", nil
		} else {
			panic(err)
		}
	} else {
		bytes, err := os.ReadFile(fpath)
		if err != nil {
			panic(err)
		}
		parse := parseGadgets(string(bytes))
		g := append(parse, *gadgets...)
		gadgets = &g
	}
	dir, file := path.Split(p)
	return dir, file, gadgets
}

func parseGadget(header string, json string, gadgets *[]Gadget) *Gadget {
	splut := strings.Split(strings.TrimPrefix(header, "#"), " ")
	if len(splut) != 2 {
		panic(errors.New(err_badHeader + header))
	}
	gadgetName := splut[0]
	if GetGadget(gadgetName, gadgets) != nil {
		panic(errors.New(err_duplicateName + gadgetName))
	}

	// we will never have enough reserved names to warrant a real search algorithm here
	for _, s := range reservedNames {
		if s == gadgetName {
			panic(err_reservedName + gadgetName)
		}
	}

	gadgetArgs, err := strconv.Atoi(splut[1])
	if err != nil {
		panic(err)
	}
	// todo: we should actually validate the json!
	return &Gadget{name: gadgetName, json: json, parametersCount: gadgetArgs}
}

func parseGadgets(s string) []Gadget {
	gadgets := []Gadget{}
	const winNewline = "\r\n"
	const normalNewline = "\n"
	var nl = normalNewline
	if strings.Contains(s, winNewline) {
		nl = winNewline
	}
	splut := strings.Split(s, nl)
	if splut[0][0] != '#' {
		panic(errors.New(err_noHeader)) // todo: this should specify which file, but that can wait
	}
	gadgetHeader := ""
	gadgetJson := ""
	for i, s := range splut {
		if s[0] == '#' {
			// this line is a header, so parse out the gadget we've built. unless this is the first gadget.
			if i != 0 {
				gadgets = append(gadgets, *parseGadget(gadgetHeader, gadgetJson, &gadgets))
			}
			gadgetHeader = s
			gadgetJson = ""
		} else {
			gadgetJson = gadgetJson + s
		}
	}
	// last gadget will never be parsed by the loop
	gadgets = append(gadgets, *parseGadget(gadgetHeader, gadgetJson, &gadgets))
	return gadgets
}

func ExpandGadget(gadget *Gadget, params []string) string {
	str := gadget.json
	for i := 0; i < gadget.parametersCount; i++ {
		str = strings.ReplaceAll(str, "%"+strconv.Itoa(i), strings.TrimSpace(params[i]))
	}
	return str
}

func GetGadget(name string, gadgets *[]Gadget) *Gadget {
	// we should actually build a map of gadgets
	for _, v := range *gadgets {
		if v.name == name {
			return &v
		}
	}
	return nil
}

func LoadGadgetsForPath(p string) *[]Gadget {
	gadgets := &builtinGadgets
	searchDir := p
	// this logic breaks if we're just starting from the working directory, but nothing it's doing needs to happen in that case anyway
	if len(p) != 0 {
		info, err := os.Stat(p)
		if err != nil {
			panic(err)
		}

		if !info.IsDir() {
			searchDir, _ = path.Split(p)
		}
	}
	var dir string
	// refactor this to not be for/break, it's gross
	for true {
		dir, _, gadgets = loadGadgetsForPath(searchDir, gadgets)
		if dir != "" {
			searchDir = dir
		} else {
			break
		}
	}
	return gadgets
}
