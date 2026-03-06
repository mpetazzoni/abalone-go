package server

import (
	"fmt"
	"math/rand"
)

var adjectives = []string{
	"swift", "brave", "clever", "bold", "calm", "dark", "eager", "fair",
	"gentle", "happy", "keen", "lively", "mighty", "noble", "proud",
	"quiet", "rapid", "sharp", "tough", "vivid", "warm", "wild", "wise",
	"ancient", "bright", "cosmic", "daring", "fierce", "golden", "hidden",
	"iron", "jade", "lunar", "mystic", "neon", "ocean", "phantom", "royal",
	"silver", "stormy", "thunder", "ultra", "velvet", "wicked", "amber",
	"blazing", "crimson", "crystal", "diamond", "emerald", "frozen",
	"granite", "hollow", "ivory", "jolly", "kinetic", "liquid", "marble",
	"nimble", "obsidian", "polar", "quartz", "rustic", "scarlet", "shadow",
	"stellar", "sunset", "tidal", "topaz", "twilight", "vapor", "volcanic",
	"winter", "zephyr", "electric", "silent", "copper", "dusty", "frosty",
}

var nouns = []string{
	"panda", "falcon", "dragon", "phoenix", "tiger", "wolf", "eagle",
	"cobra", "shark", "raven", "lion", "bear", "fox", "hawk", "viper",
	"comet", "nebula", "quasar", "pulsar", "nova", "meteor", "orbit",
	"castle", "tower", "fortress", "temple", "citadel", "beacon", "bridge",
	"canyon", "crater", "glacier", "island", "lagoon", "mesa", "oasis",
	"ravine", "summit", "valley", "volcano", "reef", "fjord", "ridge",
	"anvil", "blade", "crown", "dagger", "forge", "hammer", "lance",
	"prism", "scepter", "shield", "spark", "stone", "sword", "torch",
	"compass", "lantern", "mirror", "oracle", "puzzle", "riddle", "scroll",
	"signal", "atlas", "cipher", "echo", "flare", "glyph", "helix",
	"inferno", "junction", "keystone", "lotus", "nexus", "onyx", "pinnacle",
	"quantum", "rapids", "sentinel", "typhoon", "vortex", "zenith",
}

func generateGameCode() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	return fmt.Sprintf("%s-%s", adj, noun)
}
