package nationalrail

import "errors"

// GetAtcoCode takes a National Rail Computer Reservation System (CRS) code
// and returns an ATCO Code if available, otherwise an error.
// Data only covers the Greater Manchester area.
// This information is derived from the RailReferences.csv file in the NaPTAN
// dataset. It changes very infrequently, so it was considered not worthwhile
// storing it in a Redis cache as we have done for other NaPTAN-derived
// datasets.
func GetAtcoCode(crsCode string) (string, error) {
	var crsToAtco = make(map[string]string)

	crsToAtco["ALT"] = "9100ALTRNHM" // Altrincham
	crsToAtco["ADK"] = "9100ARDWICK" // Ardwick
	crsToAtco["ABY"] = "9100ASHBRYS" // Ashburys
	crsToAtco["AHN"] = "9100ASHONUL" // Ashton-under-Lyne
	crsToAtco["ATN"] = "9100ATHERTN" // Atherton
	crsToAtco["BLV"] = "9100BLLVUE"  // Belle Vue
	crsToAtco["BLK"] = "9100BLRD"    // Blackrod
	crsToAtco["BON"] = "9100BOLTON"  // Bolton
	crsToAtco["BML"] = "9100BMHL"    // Bramhall
	crsToAtco["BDY"] = "9100BREDBRY" // Bredbury
	crsToAtco["BNT"] = "9100BRNGTN"  // Brinnington
	crsToAtco["BDB"] = "9100BRBM"    // Broadbottom
	crsToAtco["BMC"] = "9100BRMCRSS" // Bromley Cross
	crsToAtco["BYN"] = "9100BRYN"    // Bryn
	crsToAtco["BNA"] = "9100BAGE"    // Burnage
	crsToAtco["CAS"] = "9100CSTL"    // Castleton
	crsToAtco["CSR"] = "9100CHASNRD" // Chassen Road
	crsToAtco["CHU"] = "9100CHDH"    // Cheadle Hulme
	crsToAtco["CLI"] = "9100CLTN"    // Clifton
	crsToAtco["DSY"] = "9100DAISYH"  // Daisy Hill
	crsToAtco["DVN"] = "9100DAVNPRT" // Davenport
	crsToAtco["DGT"] = "9100MNCRDGT" // Deansgate
	crsToAtco["DTN"] = "9100DNTON"   // Denton
	crsToAtco["EDY"] = "9100EDIDBRY" // East Didsbury
	crsToAtco["ECC"] = "9100ECCLES"  // Eccles
	crsToAtco["FRF"] = "9100FRFD"    // Fairfield
	crsToAtco["FLI"] = "9100FLIXTON" // Flixton
	crsToAtco["FLF"] = "9100FLWRYFD" // Flowery Field
	crsToAtco["GST"] = "9100GATHRST" // Gathurst
	crsToAtco["GTY"] = "9100GATLEY"  // Gatley
	crsToAtco["GDL"] = "9100GODLY"   // Godley
	crsToAtco["GTO"] = "9100GORTON"  // Gorton
	crsToAtco["GNF"] = "9100GFLD"    // Greenfield
	crsToAtco["GUI"] = "9100GIDB"    // Guide Bridge
	crsToAtco["HGF"] = "9100HAGFOLD" // Hag Fold
	crsToAtco["HAL"] = "9100HALE"    // Hale
	crsToAtco["HID"] = "9100HITW"    // Hall i'th' Wood
	crsToAtco["HTY"] = "9100HATRSLY" // Hattersley
	crsToAtco["HAZ"] = "9100HAZL"    // Hazel Grove
	crsToAtco["HDG"] = "9100HLDG"    // Heald Green
	crsToAtco["HTC"] = "9100HTCP"    // Heaton Chapel
	crsToAtco["HIN"] = "9100HINDLEY" // Hindley
	crsToAtco["HWI"] = "9100HORWICH" // Horwich Parkway
	crsToAtco["HUP"] = "9100HMPHRYP" // Humphrey Park
	crsToAtco["HYC"] = "9100HYDEC"   // Hyde Central
	crsToAtco["HYT"] = "9100HYDEN"   // Hyde North
	crsToAtco["INC"] = "9100INCE"    // Ince
	crsToAtco["IRL"] = "9100IRLAM"   // Irlam
	crsToAtco["KSL"] = "9100KEARSLY" // Kearsley
	crsToAtco["LVM"] = "9100LVHM"    // Levenshulme
	crsToAtco["LTL"] = "9100LITLBRO" // Littleborough
	crsToAtco["LOT"] = "9100LOSTCKP" // Lostock
	crsToAtco["MIA"] = "9100MNCRIAP" // Manchester Airport
	crsToAtco["MCO"] = "9100MNCROXR" // Manchester Oxford Road
	crsToAtco["MAN"] = "9100MNCRPIC" // Manchester Piccadilly
	crsToAtco["MUF"] = "9100MNCRUFG" // Manchester United
	crsToAtco["MCV"] = "9100MNCRVIC" // Manchester Victoria
	crsToAtco["MPL"] = "9100MARPLE"  // Marple
	crsToAtco["MAU"] = "9100MLDTHRD" // Mauldeth Road
	crsToAtco["MDL"] = "9100MDWD"    // Middlewood
	crsToAtco["MIH"] = "9100MLSHILL" // Mills Hill
	crsToAtco["MSD"] = "9100MRSD"    // Moorside
	crsToAtco["MSS"] = "9100MSGT"    // Moses Gate
	crsToAtco["MSL"] = "9100MOSSLEY" // Mossley
	crsToAtco["MSO"] = "9100MSTN"    // Moston
	crsToAtco["NVR"] = "9100NAVGTNR" // Navigation Road
	crsToAtco["NWN"] = "9100NWTH"    // Newton for Hyde
	crsToAtco["ORR"] = "9100ORRELL"  // Orrell
	crsToAtco["PAT"] = "9100PTRCRFT" // Patricroft
	crsToAtco["PEM"] = "9100PBRT"    // Pemberton
	crsToAtco["RDN"] = "9100REDISHN" // Reddish North
	crsToAtco["RDS"] = "9100REDISHS" // Reddish South
	crsToAtco["RCD"] = "9100RCHDALE" // Rochdale
	crsToAtco["RML"] = "9100ROMILEY" // Romiley
	crsToAtco["RSH"] = "9100ROHL"    // Rose Hill Marple
	crsToAtco["RRB"] = "9100RYDRBRW" // Ryder Brow
	crsToAtco["SFD"] = "9100SLFDORD" // Salford Central
	crsToAtco["SLD"] = "9100SLFDCT"  // Salford Crescent
	crsToAtco["SMB"] = "9100SMBG"    // Smithy Bridge
	crsToAtco["SYB"] = "9100SBYD"    // Stalybridge
	crsToAtco["SPT"] = "9100STKP"    // Stockport
	crsToAtco["SRN"] = "9100STRINES" // Strines
	crsToAtco["SNN"] = "9100SWNT"    // Swinton
	crsToAtco["TRA"] = "9100TRFDPK"  // Trafford Park
	crsToAtco["URM"] = "9100URMSTON" // Urmston
	crsToAtco["WKD"] = "9100WALKDEN" // Walkden
	crsToAtco["WHG"] = "9100WSTHOTN" // Westhoughton
	crsToAtco["WGN"] = "9100WIGANNW" // Wigan North Western
	crsToAtco["WGW"] = "9100WIGANWL" // Wigan Wallgate
	crsToAtco["WLY"] = "9100WOODLEY" // Woodley
	crsToAtco["WSR"] = "9100WMOR"    // Woodsmoor

	if crsToAtco[crsCode] == "" {
		return "", errors.New("CRS Code is invalid")
	}

	return crsToAtco[crsCode], nil
}
