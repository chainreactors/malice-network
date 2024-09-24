package certs

import (
	"crypto/x509/pkix"
	"fmt"
	insecureRand "math/rand"
	"strings"

	"github.com/chainreactors/malice-network/helper/codenames"
)

var (
	// Finally found a good use for Github Co-Pilot!
	// Country -> State -> Localities -> Street Addresses
	subjects = map[string]map[string]map[string][]string{
		"US": {
			"Arizona": {
				"Phoenix":    {"123 Main St", "456 Elm St"},
				"Mesa":       {"789 Oak St", "101 Pine St"},
				"Scottsdale": {"202 Maple St", "303 Cedar St"},
				"Chandler":   {"404 Birch St", "505 Walnut St"},
			},
			"California": {
				"San Francisco":       {"1000 Market St", "2000 Mission St", "3000 Broadway St"},
				"Oakland":             {"4000 Grand Ave", "5000 Lakeshore Ave"},
				"Berkeley":            {"123 University Ave", "456 Shattuck Ave"},
				"Palo Alto":           {"789 Main St", "101 Elm St"},
				"Los Angeles":         {"202 Hollywood Blvd", "303 Sunset Blvd"},
				"San Diego":           {"404 Beach Ave", "505 Ocean Blvd"},
				"San Jose":            {"606 Market St", "707 First St"},
				"Sunnyvale":           {"808 Sunny St", "909 Cloudy St"},
				"Santa Clara":         {"1010 Tech Ave", "1111 Silicon Blvd"},
				"Mountain View":       {"1212 Mountain Blvd", "1313 Valley St"},
				"San Mateo":           {"1414 Bay Ave", "1515 Shoreline Blvd"},
				"Redwood City":        {"1616 Sequoia St", "1717 Redwood Rd"},
				"Menlo Park":          {"1818 Oak St", "1919 Willow Ave"},
				"San Bruno":           {"2020 Skyline Blvd", "2121 Airport Rd"},
				"San Carlos":          {"2222 Laurel St", "2323 Pine Ave"},
				"San Leandro":         {"2424 Broad St", "2525 Marina Blvd"},
				"San Rafael":          {"2626 Fourth St", "2727 Lincoln Ave"},
				"San Ramon":           {"2828 Main St", "2929 Canyon Rd"},
				"Santa Monica":        {"3030 Ocean Ave", "3131 Beach Blvd"},
				"Santa Rosa":          {"3232 Vine St", "3333 Rose Ave"},
				"South San Francisco": {"3434 Grand Ave", "3535 Park St"},
			},
			"Colorado": {
				"Denver":       {"123 Downtown St", "456 Mountain Ave", "789 Riverwalk Dr", "101 Cherry Creek Blvd", "202 Park Lane", "303 Sunset Blvd", "404 Broadway St", "505 Lakeview Dr"},
				"Boulder":      {"606 Pearl St", "707 University Ave", "808 Canyon Blvd", "909 Walnut St", "1010 Pine St", "1111 Arapahoe Ave", "1212 Broadway Ave", "1313 Spruce St"},
				"Aurora":       {"1414 Aurora Blvd", "1515 Sunset Rd", "1616 East Ave", "1717 Park Pl", "1818 Elm St", "1919 Oak St", "2020 Main St", "2121 Garden Ave"},
				"Fort Collins": {"2222 College Ave", "2323 Old Town Rd", "2424 Mountain Ave", "2525 Main St", "2626 Riverside Dr", "2727 Lakeview Blvd", "2828 Pine St", "2929 Cherry Creek Dr"},
			},
			"Connecticut": {
				"New Haven":  {"303 Elm St", "404 Orange Ave", "505 Pine St", "606 Maple Ave", "707 Grove St", "808 Cedar Ave", "909 Cherry St", "1010 Walnut Blvd"},
				"Bridgeport": {"1111 Main St", "1212 Harbor Dr", "1313 Market St", "1414 Waterfront Ave", "1515 Ocean Blvd", "1616 Park Ave", "1717 Riverside Dr", "1818 Sunset Blvd"},
				"Stamford":   {"1919 Park Ave", "2020 Riverside Blvd", "2121 Center St", "2222 Maple Dr", "2323 Lakefront Ave", "2424 Elm St", "2525 Broadway Blvd", "2626 Pine St"},
				"Norwalk":    {"2727 Wall St", "2828 Beach Blvd", "2929 Ocean Ave", "3030 Harbor Dr", "3131 Maple Blvd", "3232 Riverfront Dr", "3333 Sunset Ave", "3434 Lake St"},
			},
			"Washington": {
				"Seattle": {"123 Pike Pl", "456 Rainier Ave", "789 Alaskan Way", "1011 Bell St", "1213 Pine St", "1415 Lakeview Dr", "1617 Cherry Creek Blvd", "1819 Harbor Dr"},
				"Tacoma":  {"2021 Market St", "2223 Ocean Blvd", "2425 Main St", "2627 Waterfront Dr", "2829 Broadway Ave", "3031 Lakefront Blvd", "3233 Sunset St", "3435 Park Blvd"},
				"Olympia": {"3637 Capitol Way", "3839 Lake St", "4041 Government Ave", "4243 Washington St", "4445 Maple Blvd", "4647 Riverfront Dr", "4849 Sunset Ave", "5051 Lakeview Dr"},
				"Spokane": {"5253 Main St", "5455 Riverfront Dr", "5657 Riverside Ave", "5859 Monroe St", "6061 Sunset Blvd", "6263 Pine St", "6465 Lakeview Dr", "6667 Broadway Ave"},
			},
			"Florida": {
				"Miami":        {"123 South Beach Blvd", "456 Ocean Dr", "789 Collins Ave", "1011 Lincoln Rd", "1213 Biscayne Blvd", "1415 Ocean Blvd", "1617 Riverside Dr", "1819 Bayshore Dr"},
				"Orlando":      {"2021 Magic Blvd", "2223 Downtown Ave", "2425 Church St", "2627 Orange Ave", "2829 Park Blvd", "3031 Sunset Dr", "3233 Main St", "3435 Riverwalk Ave"},
				"Tampa":        {"3637 Bayshore Blvd", "3839 Riverwalk Dr", "4041 Channelside Dr", "4243 Harbor Island Blvd", "4445 Ocean Blvd", "4647 Lakefront Dr", "4849 Sunset St", "5051 Park Ave"},
				"Jacksonville": {"5253 Main St", "5455 Riverfront Dr", "5657 Riverside Ave", "5859 Market St", "6061 Ocean Blvd", "6263 Lakeview Dr", "6465 Sunset Blvd", "6667 Broadway Ave"},
			},
			"Illinois": {
				"Chicago":    {"123 Michigan Ave", "456 State St", "789 Wacker Dr", "1011 Lake Shore Dr", "1213 Magnificent Mile", "1415 LaSalle St", "1617 Clark St", "1819 Randolph St"},
				"Aurora":     {"2021 Broadway St", "2223 Fox Valley Dr", "2425 Riverwalk Ave", "2627 Main St", "2829 Lakefront Dr", "3031 Cherry Creek Blvd", "3233 Riverside Dr", "3435 Sunset Blvd"},
				"Naperville": {"3637 Riverwalk Ave", "3839 Main St", "4041 Downtown Blvd", "4243 Aurora Ave", "4445 Lakefront Dr", "4647 Park Blvd", "4849 Sunset St", "5051 Cherry Creek Blvd"},
				"Peoria":     {"5253 River Rd", "5455 Downtown Blvd", "5657 Main St", "5859 Water St", "6061 Pine St", "6263 Lakeview Dr", "6465 Sunset Blvd", "6667 Broadway Ave"},
			},
			"Indiana": {
				"Indianapolis": {"123 Monument Cir", "456 Speedway Blvd", "789 Meridian St", "1011 Capitol Ave", "1213 Circle Dr", "1415 Washington St", "1617 Pennsylvania St", "1819 Market St"},
				"Fort Wayne":   {"2021 Riverfront Dr", "2223 Summit St", "2425 Main St", "2627 Broadway Ave", "2829 Lakefront Blvd", "3031 Cherry Creek Dr", "3233 Riverside Dr", "3435 Sunset St"},
				"Evansville":   {"3637 Main St", "3839 Riverside Dr", "4041 Downtown Blvd", "4243 Sunset Ave", "4445 Lakefront Dr", "4647 Park Blvd", "4849 Cherry Creek Blvd", "5051 Ocean Blvd"},
				"South Bend":   {"5253 College Ave", "5455 Downtown Blvd", "5657 Main St", "5859 Broadway Ave", "6061 Lakefront Dr", "6263 Sunset Blvd", "6465 Cherry Creek Blvd", "6667 Park Ave"},
			},
			"Massachusetts": {
				"Boston":      {"123 Beacon St", "456 Commonwealth Ave", "789 Newbury St", "1011 Boylston St", "1213 Tremont St", "1415 Huntington Ave", "1617 Cambridge St", "1819 Brookline Ave"},
				"Worcester":   {"2021 Main St", "2223 Front St", "2425 Elm St", "2627 Grove St", "2829 Park Ave", "3031 Lake St", "3233 Riverside Dr", "3435 Sunset Blvd"},
				"Springfield": {"3637 Main St", "3839 Broadway St", "4041 Park Ave", "4243 Elm St", "4445 Maple Ave", "4647 Grove St", "4849 Riverside Dr", "5051 Sunset Blvd"},
				"Lowell":      {"5253 Central St", "5455 Riverfront Dr", "5657 Downtown Blvd", "5859 Park Ave", "6061 Main St", "6263 Elm St", "6465 Maple Ave", "6667 Sunset Blvd"},
			},
			"Michigan": {
				"Detroit":          {"123 Woodward Ave", "456 Jefferson Ave", "789 Michigan Ave", "1011 Gratiot Ave", "1213 Cass Ave", "1415 Grand River Ave", "1617 Fort St", "1819 Lafayette Blvd"},
				"Grand Rapids":     {"2021 Division Ave", "2223 Fulton St", "2425 Monroe Ave", "2627 Wealthy St", "2829 Bridge St", "3031 Lake Dr", "3233 Cherry St", "3435 Michigan St"},
				"Warren":           {"3637 Van Dyke Ave", "3839 Hoover Ave", "4041 Mound Rd", "4243 Ryan Rd", "4445 12 Mile Rd", "4647 13 Mile Rd", "4849 Schoenherr Rd", "5051 Dequindre Rd"},
				"Sterling Heights": {"5253 Dodge Park Rd", "5455 Mound Rd", "5657 Utica Rd", "5859 Ryan Rd", "6061 Hayes Rd", "6263 Van Dyke Ave", "6465 Schoenherr Rd", "6667 15 Mile Rd"},
			},
			"Minnesota": {
				"Minneapolis": {"123 Hennepin Ave", "456 Nicollet Mall", "789 Washington Ave", "1011 Marquette Ave", "1213 1st Ave", "1415 2nd Ave", "1617 3rd Ave", "1819 4th Ave"},
				"Saint Paul":  {"2021 Grand Ave", "2223 University Ave", "2425 Snelling Ave", "2627 Selby Ave", "2829 Summit Ave", "3031 Randolph Ave", "3233 Marshall Ave", "3435 Grandview Blvd"},
				"Bloomington": {"3637 Old Shakopee Rd", "3839 Lyndale Ave", "4041 France Ave", "4243 Normandale Blvd", "4445 American Blvd", "4647 West 86th St", "4849 Penn Ave", "5051 Portland Ave"},
				"Plymouth":    {"5253 County Rd 101", "5455 County Rd 6", "5657 Vicksburg Ln", "5859 Northwest Blvd", "6061 Vinewood Ln", "6263 Fernbrook Ln", "6465 Lancaster Ln", "6667 Walnut Grove Ln"},
			},
			"New Jersey": {
				"Newark":      {"123 Broad St", "456 Market St", "789 Halsey St", "1011 University Ave", "1213 Raymond Blvd", "1415 Park Pl", "1617 Ferry St", "1819 Springfield Ave"},
				"Jersey City": {"2021 Hudson St", "2223 Bergen Ave", "2425 Kennedy Blvd", "2627 Montgomery St", "2829 Sip Ave", "3031 Summit Ave", "3233 Ocean Ave", "3435 Central Ave"},
				"Paterson":    {"3637 Main St", "3839 Broadway", "4041 Park Ave", "4243 Market St", "4445 Clinton St", "4647 Bridge St", "4849 Union Ave", "5051 McBride Ave"},
				"Elizabeth":   {"5253 Broad St", "5455 Elm St", "5657 1st Ave", "5859 2nd Ave", "6061 3rd Ave", "6263 4th Ave", "6465 5th Ave", "6667 6th Ave"},
			},
			"New York": {
				"New York":  {"123 Broadway", "456 Wall St", "789 Madison Ave", "1011 5th Ave", "1213 Park Ave", "1415 Lexington Ave", "1617 3rd Ave", "1819 1st Ave"},
				"Buffalo":   {"2021 Main St", "2223 Elmwood Ave", "2425 Hertel Ave", "2627 Delaware Ave", "2829 Niagara St", "3031 Amherst St", "3233 Grant St", "3435 Fillmore Ave"},
				"Rochester": {"3637 Monroe Ave", "3839 South Ave", "4041 Alexander St", "4243 East Ave", "4445 Goodman St", "4647 Park Ave", "4849 University Ave", "5051 Mount Hope Ave"},
				"Yonkers":   {"5253 Broadway", "5455 Central Park Ave", "5657 Riverdale Ave", "5859 McLean Ave", "6061 Nepperhan Ave", "6263 Saw Mill River Rd", "6465 Warburton Ave", "6667 Ridge Hill Blvd"},
			},
			"North Carolina": {
				"Charlotte":     {"123 Tryon St", "456 Trade St", "789 College St", "1011 Church St", "1213 Brevard St", "1415 Mint St", "1617 Davidson St", "1819 Caldwell St"},
				"Raleigh":       {"2021 Hillsborough St", "2223 Fayetteville St", "2425 Wilmington St", "2627 Blount St", "2829 Salisbury St", "3031 Person St", "3233 Glenwood Ave", "3435 Lake Boone Trail"},
				"Greensboro":    {"3637 Elm St", "3839 Market St", "4041 Church St", "4243 Battleground Ave", "4445 Gate City Blvd", "4647 High Point Rd", "4849 Summit Ave", "5051 Spring Garden St"},
				"Winston-Salem": {"5253 Trade St", "5455 Liberty St", "5657 Main St", "5859 Broad St", "6061 Cherry St", "6263 Elm St", "6465 Oak St", "6667 Walnut St"},
			},
			"Ohio": {
				"Columbus":   {"123 High St", "456 Broad St", "789 Gay St", "1011 Main St", "1213 Front St", "1415 Spring St", "1617 Long St", "1819 Mound St"},
				"Cleveland":  {"2021 Euclid Ave", "2223 Superior Ave", "2425 Prospect Ave", "2627 Carnegie Ave", "2829 Lakeside Ave", "3031 East Blvd", "3233 West Blvd", "3435 Park Ave"},
				"Cincinnati": {"3637 Vine St", "3839 Main St", "4041 Walnut St", "4243 Elm St", "4445 Sycamore St", "4647 Broadway St", "4849 Liberty St", "5051 Gilbert Ave"},
				"Toledo":     {"5253 Summit St", "5455 Monroe St", "5657 Broadway St", "5859 Adams St", "6061 Michigan St", "6263 Jefferson Ave", "6465 Cherry St", "6667 Erie St"},
			},
		},
		"CA": {
			"Alberta": {
				"Calgary":       {"1234 1st Ave", "5678 2nd Ave"},
				"Edmonton":      {"9101 3rd Ave", "1121 4th Ave"},
				"Red Deer":      {"3141 5th Ave", "5161 6th Ave"},
				"Fort McMurray": {"7181 7th Ave", "9191 8th Ave"},
			},
			"British Columbia": {
				"Vancouver": {"123 Main St", "456 Granville St", "789 Robson St", "1011 Burrard St", "1213 Hastings St", "1415 Georgia St", "1617 Cambie St", "1819 Broadway St"},
				"Victoria":  {"2021 Government St", "2223 Douglas St", "2425 Yates St", "2627 Fort St", "2829 Pandora Ave", "3031 Johnson St", "3233 Cook St", "3435 View St"},
				"Kelowna":   {"3637 Bernard Ave", "3839 Harvey Ave", "4041 Pandosy St", "4243 Richter St", "4445 Water St", "4647 Ellis St", "4849 Leon Ave", "5051 KLO Rd"},
				"Richmond":  {"5253 No. 3 Rd", "5455 Westminster Hwy", "5657 Bridgeport Rd", "5859 Cambie Rd", "6061 Blundell Rd", "6263 Steveston Hwy", "6465 Granville Ave", "6667 Gilbert Rd"},
			},
			"Manitoba": {
				"Winnipeg":           {"123 Portage Ave", "456 Main St", "789 Broadway St", "1011 Pembina Hwy", "1213 Donald St", "1415 Osborne St", "1617 Corydon Ave", "1819 Henderson Hwy"},
				"Brandon":            {"2021 Victoria Ave", "2223 Rosser Ave", "2425 18th St", "2627 Princess Ave", "2829 10th St", "3031 Lorne Ave", "3233 Richmond Ave", "3435 Van Horne Ave"},
				"Thompson":           {"3637 Cree Rd", "3839 Selkirk Ave", "4041 Station Rd", "4243 Princeton Dr", "4445 Deerwood Dr", "4647 Mystery Lake Rd", "4849 Lynx Cres", "5051 Nickel Rd"},
				"Portage la Prairie": {"5253 Saskatchewan Ave", "5455 Crescent Rd", "5657 Royal Rd", "5859 Tupper St", "6061 5th Ave", "6263 2nd St", "6465 Crescent Lake Rd", "6667 18th St"},
			},
			"New Brunswick": {
				"Fredericton": {"123 Queen St", "456 King St", "789 Brunswick St", "1011 Regent St", "1213 York St", "1415 Prospect St", "1617 St. Anne's Point Dr", "1819 Forest Hill Rd"},
				"Moncton":     {"2021 Main St", "2223 Mountain Rd", "2425 St. George St", "2627 Vaughan Harvey Blvd", "2829 Edinburgh Dr", "3031 Salisbury Rd", "3233 Elmwood Dr", "3435 Ryan Rd"},
				"Saint John":  {"3637 Union St", "3839 Princess St", "4041 King St", "4243 Rothesay Ave", "4445 Lansdowne Ave", "4647 Bridge St", "4849 Metcalf St", "5051 Somerset St"},
				"Dieppe":      {"5253 Champlain St", "5455 Paul St", "5657 Amirault St", "5859 Gauvin Rd", "6061 Dieppe Blvd", "6263 Mathieu St", "6465 Chartersville Rd", "6667 Melanson Rd"},
			},
			"Newfoundland and Labrador": {
				"St. John's":           {"123 Water St", "456 Duckworth St", "789 Harbour Dr", "1011 King's Rd", "1213 Military Rd", "1415 Portugal Cove Rd", "1617 Logy Bay Rd", "1819 Topsail Rd"},
				"Mount Pearl":          {"2021 Old Placentia Rd", "2223 Topsail Rd", "2425 Commonwealth Ave", "2627 Olympic Dr", "2829 Park Ave", "3031 St. David's Ave", "3233 Smallwood Dr", "3435 Ruth Ave"},
				"Conception Bay South": {"3637 CBS Hwy", "3839 Robert French Mem Pkwy", "4041 Topsail Rd", "4243 Legion Rd", "4445 Foxtrap Access Rd", "4647 Country Rd", "4849 Scenic Rd", "5051 Mad Rock Rd"},
				"Paradise":             {"5253 Karwood Dr", "5455 Topsail Rd", "5657 McNamara Dr", "5859 St. Thomas Line", "6061 Orca Dr", "6263 Carlisle Dr", "6465 Elders Ridge Rd", "6667 Indian Meal Line"},
			},
		},
		"JP": {
			"Aichi": {
				"Nagoya":  {"123-4567 Aza Kitayama", "789-0123 Aza Minamiyama", "456-7890 Aza Nishiki", "098-7654 Aza Meieki"},
				"Kasugai": {"345-6789 Aza Nishiyama", "901-2345 Aza Higashiyama", "567-8901 Aza Kitashinagawa", "234-5678 Aza Higashiyama"},
				"Okazaki": {"543-2109 Aza Asahimachi", "987-6543 Aza Nakagawa", "321-0987 Aza Sakuramachi", "765-4321 Aza Minamiyama"},
				"Handa":   {"678-9012 Aza Kameido", "345-6789 Aza Nishiyama", "890-1234 Aza Uchikoshi", "567-8901 Aza Nishio"},
			},
			"Chiba": {
				"Chiba":     {"1-2-3 Yayoi, Inage-ku", "4-5-6 Hanamigawa, Chuo-ku", "7-8-9 Wakaba, Midori-ku", "9-8-7 Inohana, Mihama-ku", "3-2-1 Yamato, Makuhari-ku", "6-5-4 Shiohama, Hanamigawa-ku", "9-7-8 Nishi, Inage-ku", "4-2-3 Kita, Chuo-ku"},
				"Kashiwa":   {"3-4-5 Sakura, Inage-ku", "6-7-8 Sumida, Chuo-ku", "1-2-3 Kanda, Midori-ku", "9-8-7 Akiba, Mihama-ku", "7-6-5 Kitami, Inage-ku", "4-3-2 Itabashi, Chuo-ku", "9-8-7 Kichijoji, Midori-ku", "2-3-4 Shinjuku, Mihama-ku"},
				"Funabashi": {"5-6-7 Kinuta, Inage-ku", "8-9-1 Ota, Chuo-ku", "2-3-4 Sugamo, Midori-ku", "7-8-9 Nakano, Mihama-ku", "1-2-3 Ikebukuro, Inage-ku", "6-7-8 Shinagawa, Chuo-ku", "3-4-5 Shinjuku, Midori-ku", "8-9-1 Ginza, Mihama-ku"},
				"Kimitsu":   {"7-8-9 Nakameguro, Inage-ku", "1-2-3 Ikebukuro, Chuo-ku", "5-6-7 Shinjuku, Midori-ku", "8-9-1 Ginza, Mihama-ku", "9-8-7 Akihabara, Inage-ku", "4-3-2 Asakusa, Chuo-ku", "1-2-3 Shibuya, Midori-ku", "6-7-8 Roppongi, Mihama-ku"},
			},
		},
	}
)

func RandomSubject(commonName string) *pkix.Name {
	country, province, locale, street := randomProvinceLocalityStreetAddress()
	return &pkix.Name{
		Organization:       randomOrganization(),
		Country:            country,
		Province:           province,
		Locality:           locale,
		StreetAddress:      street,
		PostalCode:         randomPostalCode(country),
		CommonName:         commonName,
		OrganizationalUnit: randomOrganization(),
	}
}

func randomPostalCode(country []string) []string {
	// 1 in `n` will include a postal code
	// From my cursory view of a few TLS certs it seems uncommon to include this
	// in the distinguished name so right now it's set to 1/5
	const postalProbability = 5

	if len(country) == 0 {
		return []string{}
	}
	switch country[0] {

	case "US":
		// American postal codes are 5 digits
		switch insecureRand.Intn(postalProbability) {
		case 0:
			return []string{fmt.Sprintf("%05d", insecureRand.Intn(90000)+1000)}
		default:
			return []string{}
		}

	case "CA":
		// Canadian postal codes are weird and include letter/number combo's
		letters := "ABHLMNKGJPRSTVYX"
		switch insecureRand.Intn(postalProbability) {
		case 0:
			letter1 := string(letters[insecureRand.Intn(len(letters))])
			letter2 := string(letters[insecureRand.Intn(len(letters))])
			if insecureRand.Intn(2) == 0 {
				letter1 = strings.ToLower(letter1)
				letter2 = strings.ToLower(letter2)
			}
			return []string{
				fmt.Sprintf("%s%d%s", letter1, insecureRand.Intn(9), letter2),
			}
		default:
			return []string{}
		}
	}
	return []string{}
}

func randomProvinceLocalityStreetAddress() ([]string, []string, []string, []string) {
	country := randomCountry()
	state := randomState(country)
	locality := randomLocality(country, state)
	streetAddress := randomStreetAddress(country, state, locality)
	return []string{country}, []string{state}, []string{locality}, []string{streetAddress}
}

func randomCountry() string {
	keys := make([]string, 0, len(subjects))
	for k := range subjects {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomState(country string) string {
	keys := make([]string, 0, len(subjects[country]))
	for k := range subjects[country] {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomLocality(country string, state string) string {
	locales := subjects[country][state]
	keys := make([]string, 0, len(locales))
	for k := range locales {
		keys = append(keys, k)
	}
	return keys[insecureRand.Intn(len(keys))]
}

func randomStreetAddress(country string, state string, locality string) string {
	addresses := subjects[country][state][locality]
	return addresses[insecureRand.Intn(len(addresses))]
}

var (
	orgSuffixes = []string{
		"",
		"",
		"co",
		"llc",
		"inc",
		"corp",
		"ltd",
		"plc",
		"inc.",
		"corp.",
		"ltd.",
		"plc.",
		"co.",
		"llc.",
		"incorporated",
		"limited",
		"corporation",
		"company",
		"incorporated",
		"limited",
		"corporation",
		"company",
	}
)

func randomOrganization() []string {
	adjective, _ := codenames.RandomAdjective()
	noun, _ := codenames.RandomNoun()
	suffix := orgSuffixes[insecureRand.Intn(len(orgSuffixes))]

	var orgName string
	switch insecureRand.Intn(8) {
	case 0:
		orgName = strings.TrimSpace(fmt.Sprintf("%s %s, %s", adjective, noun, suffix))
	case 1:
		orgName = strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s %s, %s", adjective, noun, suffix)))
	case 2:
		orgName = strings.TrimSpace(fmt.Sprintf("%s, %s", noun, suffix))
	case 3:
		orgName = strings.TrimSpace(strings.Title(fmt.Sprintf("%s %s, %s", adjective, noun, suffix)))
	case 4:
		orgName = strings.TrimSpace(strings.Title(fmt.Sprintf("%s %s", adjective, noun)))
	case 5:
		orgName = strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s %s", adjective, noun)))
	case 6:
		orgName = strings.TrimSpace(strings.Title(fmt.Sprintf("%s", noun)))
	case 7:
		noun2, _ := codenames.RandomNoun()
		orgName = strings.TrimSpace(strings.ToLower(fmt.Sprintf("%s-%s", noun, noun2)))
	default:
		orgName = ""
	}

	return []string{orgName}
}
