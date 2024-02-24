package parse

import (
	"strconv"

	"github.com/df-mc/dragonfly/server/world"
)

// KeyType is my enumeration for key types I've encountered so far.
type KeyType int

const (
	KeyTypeUnknown KeyType = iota
	KeyTypePlayerServer
	KeyTypePlayer
	KeyTypeMap
	KeyTypeVillageDwellers
	KeyTypeVillageInfo
	KeyTypeVillagePlayers
	KeyTypeVillagePOI
	KeyTypeVillageOverworldDwellers
	KeyTypeVillageOverworldInfo
	KeyTypeVillageOverworldPlayers
	KeyTypeVillageOverworldPOI
	KeyTypeAutonomousEntities
	KeyTypeBiomeData
	KeyTypeLevelChunkMetaDataDictionary
	KeyTypeNether
	KeyTypeOverworld
	KeyTypeTheEnd
	KeyTypeMobevents
	KeyTypePortals
	KeyTypeSchedulerWT
	KeyTypeScoreboard
	KeyTypeActorprefix
	KeyTypeDigp
	KeyTypeLocalPlayer         // Not seen in my data
	KeyTypeGameFlatworldLayers // Not seen in my data
	KeyTypeStructureTemplate   // Not seen in my data
	KeyTypeTickingArea         // Not seen in my data

	KeyTypeOverworldData3D
	KeyTypeOverworldVersion
	KeyTypeOverworldData2D
	KeyTypeOverworldData2DLegacy
	KeyTypeOverworldSubChunkPrefix
	KeyTypeOverworldLegacyTerrain
	KeyTypeOverworldBlockEntity
	KeyTypeOverworldEntity
	KeyTypeOverworldPendingTicks
	KeyTypeOverworldLegacyBlockExtraData
	KeyTypeOverworldBiomeState
	KeyTypeOverworldFinalizedState
	KeyTypeOverworldConversionData
	KeyTypeOverworldBorderBlocks
	KeyTypeOverworldHardcodedSpawners
	KeyTypeOverworldRandomTicks
	KeyTypeOverworldCheckSums
	KeyTypeOverworldGenerationSeed
	KeyTypeOverworldGeneratedPreCavesAndCliffsBlending // Not used
	KeyTypeOverworldBlendingBiomeHeight                // Not used
	KeyTypeOverworldMetaDataHash
	KeyTypeOverworldBlendingData
	KeyTypeOverworldActorDigestVersion
	KeyTypeOverworldLegacyVersion

	KeyTypeNetherData3D
	KeyTypeNetherVersion
	KeyTypeNetherData2D
	KeyTypeNetherData2DLegacy
	KeyTypeNetherSubChunkPrefix
	KeyTypeNetherLegacyTerrain
	KeyTypeNetherBlockEntity
	KeyTypeNetherEntity
	KeyTypeNetherPendingTicks
	KeyTypeNetherLegacyBlockExtraData
	KeyTypeNetherBiomeState
	KeyTypeNetherFinalizedState
	KeyTypeNetherConversionData
	KeyTypeNetherBorderBlocks
	KeyTypeNetherHardcodedSpawners
	KeyTypeNetherRandomTicks
	KeyTypeNetherCheckSums
	KeyTypeNetherGenerationSeed
	KeyTypeNetherGeneratedPreCavesAndCliffsBlending // Not used
	KeyTypeNetherBlendingBiomeHeight                // Not used
	KeyTypeNetherMetaDataHash
	KeyTypeNetherBlendingData
	KeyTypeNetherActorDigestVersion
	KeyTypeNetherLegacyVersion

	KeyTypeEndData3D
	KeyTypeEndVersion
	KeyTypeEndData2D
	KeyTypeEndData2DLegacy
	KeyTypeEndSubChunkPrefix
	KeyTypeEndLegacyTerrain
	KeyTypeEndBlockEntity
	KeyTypeEndEntity
	KeyTypeEndPendingTicks
	KeyTypeEndLegacyBlockExtraData
	KeyTypeEndBiomeState
	KeyTypeEndFinalizedState
	KeyTypeEndConversionData
	KeyTypeEndBorderBlocks
	KeyTypeEndHardcodedSpawners
	KeyTypeEndRandomTicks
	KeyTypeEndCheckSums
	KeyTypeEndGenerationSeed
	KeyTypeEndGeneratedPreCavesAndCliffsBlending // Not used
	KeyTypeEndBlendingBiomeHeight                // Not used
	KeyTypeEndMetaDataHash
	KeyTypeEndBlendingData
	KeyTypeEndActorDigestVersion
	KeyTypeEndLegacyVersion
)

// KeyTypeToString maps KeyType to its name.
var KeyTypeToString = map[KeyType]string{
	KeyTypeUnknown:                                     "KeyTypeUnknown",
	KeyTypePlayerServer:                                "KeyTypePlayerServer",
	KeyTypePlayer:                                      "KeyTypePlayer",
	KeyTypeMap:                                         "KeyTypeMap",
	KeyTypeVillageDwellers:                             "KeyTypeVillageDwellers",
	KeyTypeVillageInfo:                                 "KeyTypeVillageInfo",
	KeyTypeVillagePlayers:                              "KeyTypeVillagePlayers",
	KeyTypeVillagePOI:                                  "KeyTypeVillagePOI",
	KeyTypeVillageOverworldDwellers:                    "KeyTypeVillageOverworldDwellers",
	KeyTypeVillageOverworldInfo:                        "KeyTypeVillageOverworldInfo",
	KeyTypeVillageOverworldPlayers:                     "KeyTypeVillageOverworldPlayers",
	KeyTypeVillageOverworldPOI:                         "KeyTypeVillageOverworldPOI",
	KeyTypeAutonomousEntities:                          "KeyTypeAutonomousEntities",
	KeyTypeBiomeData:                                   "KeyTypeBiomeData",
	KeyTypeLevelChunkMetaDataDictionary:                "KeyTypeLevelChunkMetaDataDictionary",
	KeyTypeNether:                                      "KeyTypeNether",
	KeyTypeOverworld:                                   "KeyTypeOverworld",
	KeyTypeTheEnd:                                      "KeyTypeTheEnd",
	KeyTypeMobevents:                                   "KeyTypeMobevents",
	KeyTypePortals:                                     "KeyTypePortals",
	KeyTypeSchedulerWT:                                 "KeyTypeSchedulerWT",
	KeyTypeScoreboard:                                  "KeyTypeScoreboard",
	KeyTypeActorprefix:                                 "KeyTypeActorprefix",
	KeyTypeDigp:                                        "KeyTypeDigp",
	KeyTypeLocalPlayer:                                 "KeyTypeLocalPlayer",
	KeyTypeGameFlatworldLayers:                         "KeyTypeGameFlatworldLayers",
	KeyTypeStructureTemplate:                           "KeyTypeStructureTemplate",
	KeyTypeTickingArea:                                 "KeyTypeTickingArea",
	KeyTypeOverworldData3D:                             "KeyTypeOverworldData3D",
	KeyTypeOverworldVersion:                            "KeyTypeOverworldVersion",
	KeyTypeOverworldData2D:                             "KeyTypeOverworldData2D",
	KeyTypeOverworldData2DLegacy:                       "KeyTypeOverworldData2DLegacy",
	KeyTypeOverworldSubChunkPrefix:                     "KeyTypeOverworldSubChunkPrefix",
	KeyTypeOverworldLegacyTerrain:                      "KeyTypeOverworldLegacyTerrain",
	KeyTypeOverworldBlockEntity:                        "KeyTypeOverworldBlockEntity",
	KeyTypeOverworldEntity:                             "KeyTypeOverworldEntity",
	KeyTypeOverworldPendingTicks:                       "KeyTypeOverworldPendingTicks",
	KeyTypeOverworldLegacyBlockExtraData:               "KeyTypeOverworldLegacyBlockExtraData",
	KeyTypeOverworldBiomeState:                         "KeyTypeOverworldBiomeState",
	KeyTypeOverworldFinalizedState:                     "KeyTypeOverworldFinalizedState",
	KeyTypeOverworldConversionData:                     "KeyTypeOverworldConversionData",
	KeyTypeOverworldBorderBlocks:                       "KeyTypeOverworldBorderBlocks",
	KeyTypeOverworldHardcodedSpawners:                  "KeyTypeOverworldHardcodedSpawners",
	KeyTypeOverworldRandomTicks:                        "KeyTypeOverworldRandomTicks",
	KeyTypeOverworldCheckSums:                          "KeyTypeOverworldCheckSums",
	KeyTypeOverworldGenerationSeed:                     "KeyTypeOverworldGenerationSeed",
	KeyTypeOverworldGeneratedPreCavesAndCliffsBlending: "KeyTypeOverworldGeneratedPreCavesAndCliffsBlending",
	KeyTypeOverworldBlendingBiomeHeight:                "KeyTypeOverworldBlendingBiomeHeight",
	KeyTypeOverworldMetaDataHash:                       "KeyTypeOverworldMetaDataHash",
	KeyTypeOverworldBlendingData:                       "KeyTypeOverworldBlendingData",
	KeyTypeOverworldActorDigestVersion:                 "KeyTypeOverworldActorDigestVersion",
	KeyTypeOverworldLegacyVersion:                      "KeyTypeOverworldLegacyVersion",
	KeyTypeNetherData3D:                                "KeyTypeNetherData3D",
	KeyTypeNetherVersion:                               "KeyTypeNetherVersion",
	KeyTypeNetherData2D:                                "KeyTypeNetherData2D",
	KeyTypeNetherData2DLegacy:                          "KeyTypeNetherData2DLegacy",
	KeyTypeNetherSubChunkPrefix:                        "KeyTypeNetherSubChunkPrefix",
	KeyTypeNetherLegacyTerrain:                         "KeyTypeNetherLegacyTerrain",
	KeyTypeNetherBlockEntity:                           "KeyTypeNetherBlockEntity",
	KeyTypeNetherEntity:                                "KeyTypeNetherEntity",
	KeyTypeNetherPendingTicks:                          "KeyTypeNetherPendingTicks",
	KeyTypeNetherLegacyBlockExtraData:                  "KeyTypeNetherLegacyBlockExtraData",
	KeyTypeNetherBiomeState:                            "KeyTypeNetherBiomeState",
	KeyTypeNetherFinalizedState:                        "KeyTypeNetherFinalizedState",
	KeyTypeNetherConversionData:                        "KeyTypeNetherConversionData",
	KeyTypeNetherBorderBlocks:                          "KeyTypeNetherBorderBlocks",
	KeyTypeNetherHardcodedSpawners:                     "KeyTypeNetherHardcodedSpawners",
	KeyTypeNetherRandomTicks:                           "KeyTypeNetherRandomTicks",
	KeyTypeNetherCheckSums:                             "KeyTypeNetherCheckSums",
	KeyTypeNetherGenerationSeed:                        "KeyTypeNetherGenerationSeed",
	KeyTypeNetherGeneratedPreCavesAndCliffsBlending:    "KeyTypeNetherGeneratedPreCavesAndCliffsBlending",
	KeyTypeNetherBlendingBiomeHeight:                   "KeyTypeNetherBlendingBiomeHeight",
	KeyTypeNetherMetaDataHash:                          "KeyTypeNetherMetaDataHash",
	KeyTypeNetherBlendingData:                          "KeyTypeNetherBlendingData",
	KeyTypeNetherActorDigestVersion:                    "KeyTypeNetherActorDigestVersion",
	KeyTypeNetherLegacyVersion:                         "KeyTypeNetherLegacyVersion",
	KeyTypeEndData3D:                                   "KeyTypeEndData3D",
	KeyTypeEndVersion:                                  "KeyTypeEndVersion",
	KeyTypeEndData2D:                                   "KeyTypeEndData2D",
	KeyTypeEndData2DLegacy:                             "KeyTypeEndData2DLegacy",
	KeyTypeEndSubChunkPrefix:                           "KeyTypeEndSubChunkPrefix",
	KeyTypeEndLegacyTerrain:                            "KeyTypeEndLegacyTerrain",
	KeyTypeEndBlockEntity:                              "KeyTypeEndBlockEntity",
	KeyTypeEndEntity:                                   "KeyTypeEndEntity",
	KeyTypeEndPendingTicks:                             "KeyTypeEndPendingTicks",
	KeyTypeEndLegacyBlockExtraData:                     "KeyTypeEndLegacyBlockExtraData",
	KeyTypeEndBiomeState:                               "KeyTypeEndBiomeState",
	KeyTypeEndFinalizedState:                           "KeyTypeEndFinalizedState",
	KeyTypeEndConversionData:                           "KeyTypeEndConversionData",
	KeyTypeEndBorderBlocks:                             "KeyTypeEndBorderBlocks",
	KeyTypeEndHardcodedSpawners:                        "KeyTypeEndHardcodedSpawners",
	KeyTypeEndRandomTicks:                              "KeyTypeEndRandomTicks",
	KeyTypeEndCheckSums:                                "KeyTypeEndCheckSums",
	KeyTypeEndGenerationSeed:                           "KeyTypeEndGenerationSeed",
	KeyTypeEndGeneratedPreCavesAndCliffsBlending:       "KeyTypeEndGeneratedPreCavesAndCliffsBlending",
	KeyTypeEndBlendingBiomeHeight:                      "KeyTypeEndBlendingBiomeHeight",
	KeyTypeEndMetaDataHash:                             "KeyTypeEndMetaDataHash",
	KeyTypeEndBlendingData:                             "KeyTypeEndBlendingData",
	KeyTypeEndActorDigestVersion:                       "KeyTypeEndActorDigestVersion",
	KeyTypeEndLegacyVersion:                            "KeyTypeEndLegacyVersion",
}

// String returns the string representation of the given KeyType.
func (kt KeyType) String() string {
	if s := KeyTypeToString[kt]; s != "" {
		return s
	}
	return "KeyTypeINVALID:%d" + strconv.Itoa(int(kt))
}

func (kt KeyType) IsChunkData() bool {
	return kt >= KeyTypeOverworldData3D && kt <= KeyTypeEndLegacyVersion
}

func (kt KeyType) IsOverworldChunkData() bool {
	return kt >= KeyTypeOverworldData3D && kt <= KeyTypeOverworldLegacyVersion
}

func (kt KeyType) IsNetherChunkData() bool {
	return kt >= KeyTypeNetherData3D && kt <= KeyTypeNetherLegacyVersion
}

func (kt KeyType) IsEndChunkData() bool {
	return kt >= KeyTypeEndData3D && kt <= KeyTypeEndLegacyVersion
}

func (kt KeyType) IsChunkDataForDimension(d world.Dimension) bool {
	switch d {
	case world.Overworld:
		return kt.IsOverworldChunkData()
	case world.Nether:
		return kt.IsNetherChunkData()
	case world.End:
		return kt.IsEndChunkData()
	}
	return false
}

func (kt KeyType) IsSubChunkPrefix() bool {
	return kt == KeyTypeOverworldSubChunkPrefix || kt == KeyTypeNetherSubChunkPrefix || kt == KeyTypeEndSubChunkPrefix
}

/*

// Dimension is an enumeration for dimensions: Overworld, Nether, End.
type Dimension int

const (
	DimensionOverworld Dimension = 0
	DimensionNether    Dimension = 1
	DimensionEnd       Dimension = 2
	DimensionNone      Dimension = -2147483648 - 1
	DimensionError     Dimension = -2147483648 - 2
)

// String returns the string representation of the given Dimension.
func (d Dimension) String() string {
	switch d {
	case DimensionOverworld:
		return "Overworld"
	case DimensionNether:
		return "Nether"
	case DimensionEnd:
		return "End"
	case DimensionNone:
		return "DimensionNone"
	case DimensionError:
		return "DimensionError"
	default:
		return "DimensionINVALID:" + strconv.Itoa(int(d))
	}
}

*/
