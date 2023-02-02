package encoders

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"crypto/rand"
	insecureRand "math/rand"
	"testing"
	"time"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
	util "github.com/bishopfox/sliver/util/encoders"
)

const (
	sampleSizeMax = 8192
)

func init() {
	insecureRand.Seed(time.Now().Unix())
}

func randomDataRandomSize(maxSize int) []byte {
	buf := make([]byte, insecureRand.Intn(maxSize))
	rand.Read(buf)
	return buf
}

func TestCompatibilityBase64(t *testing.T) {
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(sampleSizeMax)
		output, _ := Base64.Encode(sample)
		data, err := implantEncoders.Base64.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into base64")
			return
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
		}

		sample2 := randomDataRandomSize(sampleSizeMax)
		output, _ = implantEncoders.Base64.Encode(sample2)
		data, err = Base64.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into base64")
			return
		}
		if !bytes.Equal(sample2, data) {
			t.Errorf("sample2 does not match returned\n%#v != %#v", sample2, data)
		}
	}
}

func TestCompatibilityBase58(t *testing.T) {
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(sampleSizeMax)
		output, _ := Base58.Encode(sample)
		data, err := implantEncoders.Base58.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into base58")
			return
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
		}

		sample2 := randomDataRandomSize(sampleSizeMax)
		output, _ = Base58.Encode(sample2)
		data, err = implantEncoders.Base58.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into base58")
			return
		}
		if !bytes.Equal(sample2, data) {
			t.Errorf("sample2 does not match returned\n%#v != %#v", sample2, data)
		}
	}
}

func TestCompatibilityBase32(t *testing.T) {
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(sampleSizeMax)
		output, _ := Base32.Encode(sample)
		data, err := implantEncoders.Base32.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into base32")
			return
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
		}

		sample2 := randomDataRandomSize(sampleSizeMax)
		output, _ = Base32.Encode(sample2)
		data, err = implantEncoders.Base32.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into base32")
			return
		}
		if !bytes.Equal(sample2, data) {
			t.Errorf("sample2 does not match returned\n%#v != %#v", sample2, data)
		}
	}
}

func TestCompatibilityEnglish(t *testing.T) {

	util.SetEnglishDictionary(getTestEnglishDictionary())

	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(sampleSizeMax)
		output, _ := English.Encode(sample)
		data, err := implantEncoders.English.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into english")
			return
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
		}

		sample2 := randomDataRandomSize(sampleSizeMax)
		output, _ = implantEncoders.English.Encode(sample2)
		data, err = English.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into english")
			return
		}
		if !bytes.Equal(sample2, data) {
			t.Errorf("sample2 does not match returned\n%#v != %#v", sample2, data)
		}
	}
}

func TestCompatibilityHex(t *testing.T) {
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(sampleSizeMax)
		output, _ := Hex.Encode(sample)
		data, err := implantEncoders.Hex.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into hex")
			return
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
		}

		sample2 := randomDataRandomSize(sampleSizeMax)
		output, _ = Hex.Encode(sample2)
		data, err = implantEncoders.Hex.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into hex")
			return
		}
		if !bytes.Equal(sample2, data) {
			t.Errorf("sample2 does not match returned\n%#v != %#v", sample2, data)
		}
	}
}

func TestCompatibilityGzip(t *testing.T) {
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(sampleSizeMax)
		output, _ := Gzip.Encode(sample)
		data, err := implantEncoders.Gzip.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into gzip")
			return
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
		}

		sample2 := randomDataRandomSize(sampleSizeMax)
		output, _ = Gzip.Encode(sample2)
		data, err = implantEncoders.Gzip.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into gzip")
			return
		}
		if !bytes.Equal(sample2, data) {
			t.Errorf("sample2 does not match returned\n%#v != %#v", sample2, data)
		}
	}
}

func TestCompatibilityPNG(t *testing.T) {
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(sampleSizeMax)
		output, _ := PNG.Encode(sample)
		data, err := implantEncoders.PNG.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into png")
			return
		}
		if !bytes.Equal(sample, data) {
			t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
		}

		sample2 := randomDataRandomSize(sampleSizeMax)
		output, _ = PNG.Encode(sample2)
		data, err = implantEncoders.PNG.Decode(output)
		if err != nil {
			t.Error("Failed to encode/decode sample data into png")
			return
		}
		if !bytes.Equal(sample2, data) {
			t.Errorf("sample2 does not match returned\n%#v != %#v", sample2, data)
		}
	}
}

func getTestEnglishDictionary() []string {
	return []string{

		// There are two words per-byte value, the server version of this
		// encoder uses a larger dictionary but we don't want to embed too
		// much raw data in the implant since Go already has large binaries.

		// Words were chosen at random for each given byte value.

		"SICCING",
		"NELUMBIUMS",
		"MICROPYLAR",
		"MARCHES",
		"EDDYING",
		"OMNIFIC",
		"CALAMUS",
		"ORGANOSOLS",
		"MELTWATERS",
		"BANNERS",
		"INDOWED",
		"WHIRLWINDS",
		"ERUCTED",
		"COPOLYMERS",
		"THICKHEADED",
		"BANTERS",
		"NEMESES",
		"REDUCIBILITIES",
		"CURDLES",
		"SUPERATOMS",
		"MAMMETS",
		"OCTAVES",
		"COMMOVE",
		"HAUNTER",
		"PODGILY",
		"EREMURI",
		"PERRIES",
		"QUINTUPLET",
		"MUCLUCS",
		"RELUMES",
		"AFFRICATING",
		"KELSONS",
		"BOSSISM",
		"SUPPORTIVE",
		"KOLKHOZ",
		"PARTICULARIZED",
		"ZITHERN",
		"KNEECAPPING",
		"ORAD",
		"HURTLES",
		"REVOTES",
		"SHUNTER",
		"SWOTTED",
		"BAGGINESSES",
		"REINFLAMING",
		"LOCO",
		"REENLARGING",
		"LOWBOYS",
		"CZAR",
		"CADDISED",
		"DEPRECIATOR",
		"SQUATLY",
		"GAMBADES",
		"PACHALIC",
		"BALMINESSES",
		"CHERNOZEMIC",
		"FAIRLEAD",
		"ALLOCATIONS",
		"RUBRICATING",
		"FORECASTLES",
		"FOREMANSHIP",
		"HEMISPHERES",
		"DRAUGHTSMAN",
		"ENTICEMENTS",
		"UNPERSUADED",
		"SEMITRAILER",
		"TEABOARD",
		"GREASEWOODS",
		"ARRANGED",
		"CHEERING",
		"ALLEGRETTOS",
		"THIOURACILS",
		"PICTURIZATIONS",
		"AXONOMETRIC",
		"WEEDLIKE",
		"DIDDLERS",
		"PENTACLE",
		"TALLAGES",
		"ADOPTIONIST",
		"GABFESTS",
		"METHADON",
		"FATALISM",
		"MELTAGES",
		"EVINCING",
		"PREGAMES",
		"CRAPOLAS",
		"MANDAMUS",
		"EARLDOMS",
		"LEUCINES",
		"SAMPHIRE",
		"FOOLFISH",
		"POOLHALL",
		"GREASERS",
		"EULOGIES",
		"ISOTACHS",
		"ANTITUSSIVE",
		"BLASTERS",
		"STEERING",
		"REHAB",
		"BUCKSHOT",
		"STARFISH",
		"DEAIR",
		"SULPHATE",
		"WISTFULNESS",
		"TEQUILAS",
		"SWEETIES",
		"ERASURES",
		"DEEDY",
		"SHELVERS",
		"FROSTBIT",
		"REVERSER",
		"GENERALIZATIONS",
		"CROAK",
		"PATHWAYS",
		"DISAPPROBATIONS",
		"PRECIPITATENESS",
		"LUSTRATE",
		"SALUTARY",
		"BORNE",
		"UNCLE",
		"SULLENLY",
		"SOPHISTICATEDLY",
		"PULED",
		"READDRESSING",
		"JARLS",
		"PLIES",
		"REUTTERS",
		"NONLIBRARIAN",
		"GEOMAGNETISM",
		"DOGHANGED",
		"HYPERSALIVATION",
		"PIERS",
		"TECHNOLOGIES",
		"GINNY",
		"ETHNOCENTRIC",
		"HIEROGLYPHIC",
		"FANTAILED",
		"QUADRICEPSES",
		"CONSERVATORSHIP",
		"MEANINGFULLY",
		"SCOUR",
		"STEREOTYPICALLY",
		"ROUES",
		"ORRIS",
		"EMBRACEOR",
		"SUBSATELLITE",
		"UNRESOLVABLE",
		"COLOMBARD",
		"SOURPUSS",
		"CENTRICAL",
		"GARNISHEE",
		"SPADEFISH",
		"VOYEURISTICALLY",
		"ANTICLING",
		"OBLIGATES",
		"AUTHENTICITY",
		"REEMBARKS",
		"FOLIATING",
		"PRACTICES",
		"DAUBINGLY",
		"BAKESHOPS",
		"CAPILLARY",
		"ABRASIONS",
		"COHOLDERS",
		"ACARID",
		"BOTCHIEST",
		"COIFFURES",
		"ACCIACCATURAS",
		"GALLANTLY",
		"TRICKLIER",
		"EASEFULLY",
		"GAMMED",
		"REFUSENIK",
		"UPREARING",
		"REIMBURSE",
		"NARCOTIZE",
		"BARRATORS",
		"BUTTERFAT",
		"REPLETION",
		"MISLIKERS",
		"JAUKED",
		"FILISTERS",
		"OVERDRIVE",
		"GRIPPIEST",
		"BEARDEDNESSES",
		"SWORDFISH",
		"ARGALS",
		"WITHOUTDOORS",
		"FOOTSTOCK",
		"STONECROP",
		"NAPROXENS",
		"PRESERVES",
		"FISSILITY",
		"SOLAND",
		"BOINGS",
		"BACKHANDER",
		"SPECIFICITIES",
		"CINQUE",
		"LISTEE",
		"RESTRIVES",
		"TODIES",
		"LECTOR",
		"VASSAL",
		"WRACKS",
		"RODENT",
		"QUASAR",
		"CHOROGRAPHIES",
		"BOSQUE",
		"AEROBRAKED",
		"SHELLY",
		"SQUALL",
		"DECATHLETE",
		"IMMUNOGENETIC",
		"MEDICATING",
		"DECENNIALS",
		"REZERO",
		"INVERT",
		"EXPLICABLE",
		"INARGUABLE",
		"OCTOSYLLABICS",
		"HYPERLIPEMIAS",
		"CARAMELIZE",
		"VICTIMOLOGIES",
		"GENTAMICIN",
		"SURRAS",
		"IMPEACHERS",
		"SCAGLIOLAS",
		"SKIMOBILED",
		"PREPROCESSING",
		"HYPERROMANTIC",
		"DESTAINING",
		"LYRIST",
		"CIGUATERAS",
		"RATCHETING",
		"SACAHUISTE",
		"CORNCRAKES",
		"MUCKRAKING",
		"INTERMOUNTAIN",
		"CARBOLIZES",
		"BESWARMING",
		"BOWDLERISE",
		"EUTHANIZED",
		"LENTIGINES",
		"SLOBBERERS",
		"HEEDING",
		"KEBBIES",
		"INJECTIONS",
		"DERMATOSES",
		"TRACHEA",
		"PLANATIONS",
		"REVOCATION",
		"APHTHAE",
		"MAUMETRIES",
		"GOOMBAH",
		"HETAIRA",
		"HARDIER",
		"GROUNDINGS",
		"HAMMING",
		"ASCOSPORES",
		"NONHOSTILE",
		"MAIHEMS",
		"LACTAMS",
		"FIREPOWERS",
		"SJAMBOK",
		"HYDROSERES",
		"DERNIER",
		"AILERON",
		"KARATES",
		"POLLINOSIS",
		"RAMMIER",
		"SCORNED",
		"MARMITE",
		"BOOTLESSLY",
		"FRISEES",
		"BUSHMEN",
		"HYSTERESIS",
		"COMMUNE",
		"GROSSULARS",
		"THRIVED",
		"FOCUSER",
		"PANTISOCRACIES",
		"ZOMBIES",
		"HIRSLES",
		"SNAKILY",
		"CASE",
		"WOORALI",
		"QUEUING",
		"ERRANTS",
		"PYRETIC",
		"AMNESTY",
		"CHALCOCITES",
		"POIKILOTHERMIC",
		"CULVERS",
		"STUDIOUSLY",
		"PARONYM",
		"MURRIES",
		"WETTISH",
		"SEMINOMADIC",
		"SHOVERS",
		"NONZERO",
		"LAZE",
		"RUNDOWN",
		"FEDERALIZES",
		"REDEVELOPED",
		"ZONULES",
		"PUPPETS",
		"NAZI",
		"ZOOLOGY",
		"JOYPOPS",
		"DECLIVITIES",
		"REASSEMBLES",
		"HORN",
		"PREDOMINATE",
		"OSCILLATING",
		"MESHUGGENER",
		"GRAVIDITIES",
		"COWS",
		"RESHUFFLING",
		"SCRABBLE",
		"AGENCIES",
		"OEDEMATA",
		"REEVALUATES",
		"FADEAWAY",
		"DRAFFIER",
		"CONFERMENTS",
		"MISDOUBTING",
		"OSTRACISING",
		"WITS",
		"FOZY",
		"AUDIBLES",
		"UNSOLDERING",
		"DACTYLOLOGY",
		"BEAMLESS",
		"ADMITTEE",
		"SANDBARS",
		"BLUECAPS",
		"DEPLUMED",
		"GENTRICE",
		"MEANINGS",
		"DEVIANCY",
		"NEUTRALNESS",
		"REJUVENATOR",
		"ROUGHNESSES",
		"SUBBASIN",
		"OMNIMODE",
		"APPENDIX",
		"CYTOSTATICS",
		"PURCHASE",
		"AUTHORED",
		"SUBTHEME",
		"AGAPE",
		"INEFFECTUALNESS",
		"GROUCHES",
		"STOWABLE",
		"EASED",
		"SUBROUTINES",
		"DOUBLEHEADER",
		"PSYCHING",
		"GUESTING",
		"SOUNDING",
		"TERRENES",
		"FRONTCOURTS",
		"KIANG",
		"UMPIRING",
		"COWORKER",
		"SOUTHWESTER",
		"HUNGOVER",
		"ANTIMONY",
		"STEADFASTNESSES",
		"OUTBOAST",
		"KEYSTONE",
		"ULTRAHOT",
		"BLIMP",
		"COUNTERVIOLENCE",
		"SLICK",
		"BRANT",
		"SPIFF",
		"UPSIZING",
		"COUNTERTENDENCY",
		"GYNECOCRATIC",
		"ORPHREYS",
		"CONTOURS",
		"MEETS",
		"JONES",
		"WATERFLOODED",
		"INCUR",
		"HYPERSALIVATION",
		"REVISORY",
		"NONDECEPTIVE",
		"DRILLABILITY",
		"OUTVAUNT",
		"OUTPRAYS",
		"PITCHFORKING",
		"MOTTE",
		"OVERBUILDING",
		"ADVANTAGE",
		"FLABBIEST",
		"CALAMARIS",
		"ROVER",
		"RETACKLED",
		"ICTERICAL",
		"YTTRIUMS",
		"INNOVATIONAL",
		"ABNEGATOR",
		"KERCHIEFS",
		"TREWS",
		"TRANSSEXUALISMS",
		"POSTCOLONIAL",
		"WORMS",
		"MESHUGGAH",
		"GEMMOLOGISTS",
		"OVERORGANIZE",
		"EPIFAUNAS",
		"DIGITIZED",
		"ELECTRICS",
		"MONOGENIC",
		"METALWORKERS",
		"APHOLATES",
		"EMULSIBLE",
		"MENTIONED",
		"REPUTABLE",
		"BLOODIEST",
		"EMBRASURE",
		"LAMINATOR",
		"INNOVATED",
		"NONGLARES",
		"CHAKRA",
		"SELFWARDS",
		"UNDERGOES",
		"MYELOGRAM",
		"RADWASTES",
		"ASSIGNERS",
		"GLYCERINS",
		"LEVITATES",
		"WORLDLING",
		"REANOINTS",
		"BLUEWOODS",
		"BIOGAS",
		"CHINAS",
		"ARCHIPELAGOES",
		"SITUATING",
		"PINEAL",
		"VIRTUOSITIES",
		"POSTPOSITION",
		"SLEIGH",
		"PHLEGM",
		"RETEAM",
		"LIERNE",
		"SENILE",
		"PRESANCTIFIED",
		"EMETIN",
		"APRIORITY",
		"ASLOPE",
		"HAZANS",
		"SITARISTS",
		"DREARY",
		"AUTOROUTE",
		"SHAVER",
		"GAILLARDIA",
		"COSTUMERY",
		"SIMLIN",
		"KITTEL",
		"UMBERS",
		"GOWANS",
		"MARKUP",
		"ALIYOS",
		"SORGHO",
		"OUTGAS",
		"UNCUTE",
		"TETANY",
		"UNWARRANTABLE",
		"PAWNOR",
		"LAPSUS",
		"GUIROS",
		"UNSCALABLE",
		"MANIACALLY",
		"DOCUMENTALIST",
		"TYRING",
		"VACATIONED",
		"OUTCOACHED",
		"SADDLERIES",
		"RACEWALKER",
		"MUNCHABLES",
		"HITCHHIKER",
		"TETRAHEDRA",
		"NONDEFENSE",
		"SIRUPS",
		"PERICYCLIC",
		"ENDOMETRIA",
		"UNSHEATHED",
		"ACCLAIM",
		"EVANGELISM",
		"SKIVVY",
		"MEDIATIONS",
		"TWISTS",
		"CRYSTALLIZING",
		"ABOLLAE",
		"STERILIZATION",
		"HIEROPHANT",
		"EPITOMISED",
		"NEIGHED",
		"JANGLED",
		"LEADMEN",
		"WITCHWEEDS",
		"ROCKSHAFTS",
		"STOMACHERS",
		"SALESWOMAN",
		"SCALENE",
		"KECKING",
		"LATENED",
		"SUBSIDISES",
		"CARDIOTHORACIC",
	}
}
