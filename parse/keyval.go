package parse

import (
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

type KeyInfo struct {
	KeyType     KeyType
	ChunkPos    world.ChunkPos
	Dimension   world.Dimension
	HasLocation bool
}

// KeyVal holds and can interpret keys and values from the leveldb
type KeyVal struct {
	Key           []byte
	Val           []byte
	cachedKeyInfo *KeyInfo
}

// NewKeyVal returns a new KeyVal struct containing clones of the
// passed key and value.
func NewKeyVal(key, val []byte) *KeyVal {
	return &KeyVal{
		Key: slices.Clone(key),
		Val: slices.Clone(val),
	}
}

// KeyType figures out and returns the KeyType of the Key of this
// KeyVal.
func (kv *KeyVal) KeyType() KeyType {
	if kv.cachedKeyInfo == nil {
		kv.KeyTypeAndChunkLocation()
	}
	return kv.cachedKeyInfo.KeyType
}

// KeyTypeAndChunkLocations figures out and returns a KeyInfo struct
// with the KeyType of the Key of this KeyVal, and optionally the
// chunk location. If it cannot figure anything out, it returns
// KeyTypeUnknown.
func (kv *KeyVal) KeyTypeAndChunkLocation() KeyInfo {
	if kv.cachedKeyInfo != nil {
		return *kv.cachedKeyInfo
	}

	// We'll write directly into the cachedKeyInfo field's value.
	res := &KeyInfo{}
	kv.cachedKeyInfo = res

	ks := string(kv.Key)
	kl := len(ks)

	if keyType, ok := fullKeyStrings[ks]; ok {
		res.KeyType = keyType
		return *res
	}

	for _, prefixSuffixAndType := range keyStringUniquePrefixesAndSuffixes {
		if strings.HasPrefix(ks, prefixSuffixAndType.prefix) && strings.HasSuffix(ks, prefixSuffixAndType.suffix) {
			res.KeyType = prefixSuffixAndType.keyType
			return *res
		}
	}

	if kl == 9 || kl == 10 || kl == 13 || kl == 14 {
		chunkPos, dimension, err := ParseSaneChunkPrefix(kv.Key)
		if err == nil {
			levelChunkTag := LevelChunkTag(kv.Key[kl&^3])
			keyType := levelChunkTag.KeyTypeForDimension(dimension)
			if keyType != KeyTypeUnknown {
				res.KeyType = keyType
				res.ChunkPos = chunkPos
				res.Dimension = dimension
				res.HasLocation = true
				return *res
			}
		}
	}

	res.KeyType = KeyTypeUnknown
	return *res
}

// fullKeyStrings maps strings that are expected to be seen as entire
// keys to their KeyType.
var fullKeyStrings = map[string]KeyType{
	"AutonomousEntities":           KeyTypeAutonomousEntities,
	"BiomeData":                    KeyTypeBiomeData,
	"LevelChunkMetaDataDictionary": KeyTypeLevelChunkMetaDataDictionary,
	"Nether":                       KeyTypeNether,
	"Overworld":                    KeyTypeOverworld,
	"TheEnd":                       KeyTypeTheEnd,
	"mobevents":                    KeyTypeMobevents,
	"portals":                      KeyTypePortals,
	"schedulerWT":                  KeyTypeSchedulerWT,
	"scoreboard":                   KeyTypeScoreboard,
	"~local_player":                KeyTypeLocalPlayer,
	"game_flatworldlayers":         KeyTypeGameFlatworldLayers,
	"structuretemplate":            KeyTypeStructureTemplate,
	"tickingarea":                  KeyTypeTickingArea,
}

// keyStringUniquePrefixes maps prefixes that uniquely identify the
// KeyType of their key to that KeyType.
var keyStringUniquePrefixes = []struct {
	prefix  string
	keyType KeyType
}{}

// keyStringUniquePrefixesAndSuffixes maps prefix/suffix pairs that
// uniquely identify the KeyType of their key to that KeyType.
var keyStringUniquePrefixesAndSuffixes = []struct {
	prefix  string
	suffix  string
	keyType KeyType
}{
	{"map_-", "", KeyTypeMap},
	{"player_server_", "", KeyTypePlayerServer},
	{"player_", "", KeyTypePlayer},
	{"actorprefix", "", KeyTypeActorprefix},
	{"digp", "", KeyTypeDigp},

	{"VILLAGE_Overworld_", "_DWELLERS", KeyTypeVillageOverworldDwellers},
	{"VILLAGE_Overworld_", "_INFO", KeyTypeVillageOverworldInfo},
	{"VILLAGE_Overworld_", "_PLAYERS", KeyTypeVillageOverworldPlayers},
	{"VILLAGE_Overworld_", "_POI", KeyTypeVillageOverworldPOI},
	{"VILLAGE_", "_DWELLERS", KeyTypeVillageDwellers},
	{"VILLAGE_", "_INFO", KeyTypeVillageInfo},
	{"VILLAGE_", "_PLAYERS", KeyTypeVillagePlayers},
	{"VILLAGE_", "_POI", KeyTypeVillagePOI},
}

// "VILLAGE_",
// "VILLAGE_Overworld_",

type tagDim = struct {
	tag LevelChunkTag
	dim int
}

func (lct LevelChunkTag) KeyTypeForDimension(d world.Dimension) KeyType {
	dimensionID, _ := world.DimensionID(d)
	if keyType, ok := levelChunkTagToKeyType[tagDim{tag: lct, dim: dimensionID}]; ok {
		return keyType
	}
	return KeyTypeUnknown
}

func (kt KeyType) LevelChunkTag() LevelChunkTag {
	if lct, ok := keyTypeToLevelChunkTag[kt]; ok {
		return lct
	}
	return LevelChunkTagUnknown
}

var levelChunkTagToKeyType = map[tagDim]KeyType{
	{LevelChunkTagUnknown, 0}: KeyTypeUnknown,
	{LevelChunkTagUnknown, 1}: KeyTypeUnknown,
	{LevelChunkTagUnknown, 2}: KeyTypeUnknown,

	{LevelChunkTagData3D, 0}:               KeyTypeOverworldData3D,
	{LevelChunkTagVersion, 0}:              KeyTypeOverworldVersion,
	{LevelChunkTagData2D, 0}:               KeyTypeOverworldData2D,
	{LevelChunkTagData2DLegacy, 0}:         KeyTypeOverworldData2DLegacy,
	{LevelChunkTagSubChunkPrefix, 0}:       KeyTypeOverworldSubChunkPrefix,
	{LevelChunkTagLegacyTerrain, 0}:        KeyTypeOverworldLegacyTerrain,
	{LevelChunkTagBlockEntity, 0}:          KeyTypeOverworldBlockEntity,
	{LevelChunkTagEntity, 0}:               KeyTypeOverworldEntity,
	{LevelChunkTagPendingTicks, 0}:         KeyTypeOverworldPendingTicks,
	{LevelChunkTagLegacyBlockExtraData, 0}: KeyTypeOverworldLegacyBlockExtraData,
	{LevelChunkTagBiomeState, 0}:           KeyTypeOverworldBiomeState,
	{LevelChunkTagFinalizedState, 0}:       KeyTypeOverworldFinalizedState,
	{LevelChunkTagConversionData, 0}:       KeyTypeOverworldConversionData,
	{LevelChunkTagBorderBlocks, 0}:         KeyTypeOverworldBorderBlocks,
	{LevelChunkTagHardcodedSpawners, 0}:    KeyTypeOverworldHardcodedSpawners,
	{LevelChunkTagRandomTicks, 0}:          KeyTypeOverworldRandomTicks,
	{LevelChunkTagCheckSums, 0}:            KeyTypeOverworldCheckSums,
	{LevelChunkTagGenerationSeed, 0}:       KeyTypeOverworldGenerationSeed,
	{LevelChunkTagMetaDataHash, 0}:         KeyTypeOverworldMetaDataHash,
	{LevelChunkTagBlendingData, 0}:         KeyTypeOverworldBlendingData,
	{LevelChunkTagActorDigestVersion, 0}:   KeyTypeOverworldActorDigestVersion,
	{LevelChunkTagLegacyVersion, 0}:        KeyTypeOverworldLegacyVersion,
	// Not used
	{LevelChunkTagGeneratedPreCavesAndCliffsBlending, 0}: KeyTypeOverworldGeneratedPreCavesAndCliffsBlending,
	{LevelChunkTagBlendingBiomeHeight, 0}:                KeyTypeOverworldBlendingBiomeHeight,

	{LevelChunkTagData3D, 1}:               KeyTypeNetherData3D,
	{LevelChunkTagVersion, 1}:              KeyTypeNetherVersion,
	{LevelChunkTagData2D, 1}:               KeyTypeNetherData2D,
	{LevelChunkTagData2DLegacy, 1}:         KeyTypeNetherData2DLegacy,
	{LevelChunkTagSubChunkPrefix, 1}:       KeyTypeNetherSubChunkPrefix,
	{LevelChunkTagLegacyTerrain, 1}:        KeyTypeNetherLegacyTerrain,
	{LevelChunkTagBlockEntity, 1}:          KeyTypeNetherBlockEntity,
	{LevelChunkTagEntity, 1}:               KeyTypeNetherEntity,
	{LevelChunkTagPendingTicks, 1}:         KeyTypeNetherPendingTicks,
	{LevelChunkTagLegacyBlockExtraData, 1}: KeyTypeNetherLegacyBlockExtraData,
	{LevelChunkTagBiomeState, 1}:           KeyTypeNetherBiomeState,
	{LevelChunkTagFinalizedState, 1}:       KeyTypeNetherFinalizedState,
	{LevelChunkTagConversionData, 1}:       KeyTypeNetherConversionData,
	{LevelChunkTagBorderBlocks, 1}:         KeyTypeNetherBorderBlocks,
	{LevelChunkTagHardcodedSpawners, 1}:    KeyTypeNetherHardcodedSpawners,
	{LevelChunkTagRandomTicks, 1}:          KeyTypeNetherRandomTicks,
	{LevelChunkTagCheckSums, 1}:            KeyTypeNetherCheckSums,
	{LevelChunkTagGenerationSeed, 1}:       KeyTypeNetherGenerationSeed,
	{LevelChunkTagMetaDataHash, 1}:         KeyTypeNetherMetaDataHash,
	{LevelChunkTagBlendingData, 1}:         KeyTypeNetherBlendingData,
	{LevelChunkTagActorDigestVersion, 1}:   KeyTypeNetherActorDigestVersion,
	{LevelChunkTagLegacyVersion, 1}:        KeyTypeNetherLegacyVersion,
	// Not used
	{LevelChunkTagGeneratedPreCavesAndCliffsBlending, 1}: KeyTypeNetherGeneratedPreCavesAndCliffsBlending,
	{LevelChunkTagBlendingBiomeHeight, 1}:                KeyTypeNetherBlendingBiomeHeight,

	{LevelChunkTagData3D, 2}:               KeyTypeEndData3D,
	{LevelChunkTagVersion, 2}:              KeyTypeEndVersion,
	{LevelChunkTagData2D, 2}:               KeyTypeEndData2D,
	{LevelChunkTagData2DLegacy, 2}:         KeyTypeEndData2DLegacy,
	{LevelChunkTagSubChunkPrefix, 2}:       KeyTypeEndSubChunkPrefix,
	{LevelChunkTagLegacyTerrain, 2}:        KeyTypeEndLegacyTerrain,
	{LevelChunkTagBlockEntity, 2}:          KeyTypeEndBlockEntity,
	{LevelChunkTagEntity, 2}:               KeyTypeEndEntity,
	{LevelChunkTagPendingTicks, 2}:         KeyTypeEndPendingTicks,
	{LevelChunkTagLegacyBlockExtraData, 2}: KeyTypeEndLegacyBlockExtraData,
	{LevelChunkTagBiomeState, 2}:           KeyTypeEndBiomeState,
	{LevelChunkTagFinalizedState, 2}:       KeyTypeEndFinalizedState,
	{LevelChunkTagConversionData, 2}:       KeyTypeEndConversionData,
	{LevelChunkTagBorderBlocks, 2}:         KeyTypeEndBorderBlocks,
	{LevelChunkTagHardcodedSpawners, 2}:    KeyTypeEndHardcodedSpawners,
	{LevelChunkTagRandomTicks, 2}:          KeyTypeEndRandomTicks,
	{LevelChunkTagCheckSums, 2}:            KeyTypeEndCheckSums,
	{LevelChunkTagGenerationSeed, 2}:       KeyTypeEndGenerationSeed,
	{LevelChunkTagMetaDataHash, 2}:         KeyTypeEndMetaDataHash,
	{LevelChunkTagBlendingData, 2}:         KeyTypeEndBlendingData,
	{LevelChunkTagActorDigestVersion, 2}:   KeyTypeEndActorDigestVersion,
	{LevelChunkTagLegacyVersion, 2}:        KeyTypeEndLegacyVersion,
	// Not used
	{LevelChunkTagGeneratedPreCavesAndCliffsBlending, 2}: KeyTypeEndGeneratedPreCavesAndCliffsBlending,
	{LevelChunkTagBlendingBiomeHeight, 2}:                KeyTypeEndBlendingBiomeHeight,
}

var keyTypeToLevelChunkTag = map[KeyType]LevelChunkTag{
	KeyTypeUnknown: LevelChunkTagUnknown,

	KeyTypeOverworldData3D:               LevelChunkTagData3D,
	KeyTypeOverworldVersion:              LevelChunkTagVersion,
	KeyTypeOverworldData2D:               LevelChunkTagData2D,
	KeyTypeOverworldData2DLegacy:         LevelChunkTagData2DLegacy,
	KeyTypeOverworldSubChunkPrefix:       LevelChunkTagSubChunkPrefix,
	KeyTypeOverworldLegacyTerrain:        LevelChunkTagLegacyTerrain,
	KeyTypeOverworldBlockEntity:          LevelChunkTagBlockEntity,
	KeyTypeOverworldEntity:               LevelChunkTagEntity,
	KeyTypeOverworldPendingTicks:         LevelChunkTagPendingTicks,
	KeyTypeOverworldLegacyBlockExtraData: LevelChunkTagLegacyBlockExtraData,
	KeyTypeOverworldBiomeState:           LevelChunkTagBiomeState,
	KeyTypeOverworldFinalizedState:       LevelChunkTagFinalizedState,
	KeyTypeOverworldConversionData:       LevelChunkTagConversionData,
	KeyTypeOverworldBorderBlocks:         LevelChunkTagBorderBlocks,
	KeyTypeOverworldHardcodedSpawners:    LevelChunkTagHardcodedSpawners,
	KeyTypeOverworldRandomTicks:          LevelChunkTagRandomTicks,
	KeyTypeOverworldCheckSums:            LevelChunkTagCheckSums,
	KeyTypeOverworldGenerationSeed:       LevelChunkTagGenerationSeed,
	KeyTypeOverworldMetaDataHash:         LevelChunkTagMetaDataHash,
	KeyTypeOverworldBlendingData:         LevelChunkTagBlendingData,
	KeyTypeOverworldActorDigestVersion:   LevelChunkTagActorDigestVersion,
	KeyTypeOverworldLegacyVersion:        LevelChunkTagLegacyVersion,
	// Not used
	KeyTypeOverworldGeneratedPreCavesAndCliffsBlending: LevelChunkTagGeneratedPreCavesAndCliffsBlending,
	KeyTypeOverworldBlendingBiomeHeight:                LevelChunkTagBlendingBiomeHeight,

	KeyTypeNetherData3D:               LevelChunkTagData3D,
	KeyTypeNetherVersion:              LevelChunkTagVersion,
	KeyTypeNetherData2D:               LevelChunkTagData2D,
	KeyTypeNetherData2DLegacy:         LevelChunkTagData2DLegacy,
	KeyTypeNetherSubChunkPrefix:       LevelChunkTagSubChunkPrefix,
	KeyTypeNetherLegacyTerrain:        LevelChunkTagLegacyTerrain,
	KeyTypeNetherBlockEntity:          LevelChunkTagBlockEntity,
	KeyTypeNetherEntity:               LevelChunkTagEntity,
	KeyTypeNetherPendingTicks:         LevelChunkTagPendingTicks,
	KeyTypeNetherLegacyBlockExtraData: LevelChunkTagLegacyBlockExtraData,
	KeyTypeNetherBiomeState:           LevelChunkTagBiomeState,
	KeyTypeNetherFinalizedState:       LevelChunkTagFinalizedState,
	KeyTypeNetherConversionData:       LevelChunkTagConversionData,
	KeyTypeNetherBorderBlocks:         LevelChunkTagBorderBlocks,
	KeyTypeNetherHardcodedSpawners:    LevelChunkTagHardcodedSpawners,
	KeyTypeNetherRandomTicks:          LevelChunkTagRandomTicks,
	KeyTypeNetherCheckSums:            LevelChunkTagCheckSums,
	KeyTypeNetherGenerationSeed:       LevelChunkTagGenerationSeed,
	KeyTypeNetherMetaDataHash:         LevelChunkTagMetaDataHash,
	KeyTypeNetherBlendingData:         LevelChunkTagBlendingData,
	KeyTypeNetherActorDigestVersion:   LevelChunkTagActorDigestVersion,
	KeyTypeNetherLegacyVersion:        LevelChunkTagLegacyVersion,
	// Not used
	KeyTypeNetherGeneratedPreCavesAndCliffsBlending: LevelChunkTagGeneratedPreCavesAndCliffsBlending,
	KeyTypeNetherBlendingBiomeHeight:                LevelChunkTagBlendingBiomeHeight,

	KeyTypeEndData3D:               LevelChunkTagData3D,
	KeyTypeEndVersion:              LevelChunkTagVersion,
	KeyTypeEndData2D:               LevelChunkTagData2D,
	KeyTypeEndData2DLegacy:         LevelChunkTagData2DLegacy,
	KeyTypeEndSubChunkPrefix:       LevelChunkTagSubChunkPrefix,
	KeyTypeEndLegacyTerrain:        LevelChunkTagLegacyTerrain,
	KeyTypeEndBlockEntity:          LevelChunkTagBlockEntity,
	KeyTypeEndEntity:               LevelChunkTagEntity,
	KeyTypeEndPendingTicks:         LevelChunkTagPendingTicks,
	KeyTypeEndLegacyBlockExtraData: LevelChunkTagLegacyBlockExtraData,
	KeyTypeEndBiomeState:           LevelChunkTagBiomeState,
	KeyTypeEndFinalizedState:       LevelChunkTagFinalizedState,
	KeyTypeEndConversionData:       LevelChunkTagConversionData,
	KeyTypeEndBorderBlocks:         LevelChunkTagBorderBlocks,
	KeyTypeEndHardcodedSpawners:    LevelChunkTagHardcodedSpawners,
	KeyTypeEndRandomTicks:          LevelChunkTagRandomTicks,
	KeyTypeEndCheckSums:            LevelChunkTagCheckSums,
	KeyTypeEndGenerationSeed:       LevelChunkTagGenerationSeed,
	KeyTypeEndMetaDataHash:         LevelChunkTagMetaDataHash,
	KeyTypeEndBlendingData:         LevelChunkTagBlendingData,
	KeyTypeEndActorDigestVersion:   LevelChunkTagActorDigestVersion,
	KeyTypeEndLegacyVersion:        LevelChunkTagLegacyVersion,
	// Not used
	KeyTypeEndGeneratedPreCavesAndCliffsBlending: LevelChunkTagGeneratedPreCavesAndCliffsBlending,
	KeyTypeEndBlendingBiomeHeight:                LevelChunkTagBlendingBiomeHeight,
}
