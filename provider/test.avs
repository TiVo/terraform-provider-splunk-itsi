{
   "type":"record",
   "name":"PgdDataResponse",
   "namespace":"com.tivo.udf.pgd.api.model",
   "doc":"A set of PGD entities for a station group at a specific version for a station group and schedule length.",
   "fields":[
      {
         "name":"isCompleteSchedule",
         "type":"boolean",
         "doc":"True if the entity represents a complete schedule, false if it represents a delta."
      },
      {
         "name":"scheduleLength",
         "type":
{
   "type": "enum",
   "name": "ScheduleLength",
   "namespace": "com.tivo.udf.pgd.api.model",
   "symbols": [
      "TWO_DAYS",
      "THREE_DAYS",
      "SEVEN_DAYS",
      "FOURTEEN_DAYS"
   ]
},
         "doc":"The ScheduleLength that the current response is for."
      },
      {
         "name":"version",
         "type":"string",
         "doc":"The version of PGD data that the response contains. Note that for deltas, only data from a certain fromVersion will be present in the response."
      },
      {
         "name":"expiresAtSeconds",
         "type":"int",
         "doc":"The moment in time when this given version will expire(UTC seconds)."
      },
      {
         "name":"stationGroup",
         "type":"string"
      },
      {
         "name":"showings",
         "type":{
            "type":"array",
            "items":
{
  "type" : "record",
  "name" : "Showing",
  "namespace" : "com.tivo.udf.pgd.api.model",
  "doc" : "A showing is the presentation of a piece of content on a specific station at a specific date and time, for a specific duration.",
  "fields" : [ {
    "name" : "version",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "\d{1,3}\.\d{1,3}(\.\d{1,3})?"
    },
    "doc" : "This is the version of the _writer's PGD API Schema_ at the time the messages was created."
  }, {
    "name" : "generated",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "\d{1,3}\.\d{1,3}(\.\d{1,3})?"
    },
    "doc" : "This is the version of the UDF PGD API Transformer that produced this message."
  }, {
    "name" : "sourceID",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "([a-zA-Z][a-zA-Z0-9_.]*)\$([a-z_][a-z0-9_.]*)(\![a-z_][a-z0-9_.]*)+\)(.+)"
    },
    "doc" : "This is the internal UDF \"Resource ID\" from which this document was generated. This can be very helpful for debugging. The format is `{entity}${primary}`, where `{entity}` is the UDF type (e.g. \"Episode\"\") and `{primary}` is the UDF Primary value (e.g. \"tivo!c!r)123\"). Note: when the primary starts with \"tivo!c!r)\", then the number that follows is the original Metadata Rovi 2.0 Content ID (that is, the sourceProgramId from the Ref Server)."
  }, {
    "name" : "showingID",
    "type" : {
      "type" : "long",
      "minExclusive" : 0
    },
    "doc" : "A 64-bit ID from the \"Rovi Content 2.0\" space, assigned by the TiVo Metadata team. Used only for Showings."
  }, {
    "name" : "contentID",
    "type" : {
      "type" : "long",
      "minExclusive" : 0,
      "hierarchy" : "/trioObjectID/"
    },
    "doc" : "The Trio Object ID for the content being shown."
  }, {
    "name" : "stationID",
    "type" : {
      "type" : "long",
      "minExclusive" : 0,
      "hierarchy" : "/trioObjectID/"
    },
    "doc" : "The Trio Object ID for the station on which this showing will appear."
  }, {
    "name" : "startTime",
    "type" : {
      "type" : "long",
      "logicalType" : "timestamp-seconds",
      "hierarchy" : "/instant/"
    },
    "doc" : "The start time for this showing, in \"epoch seconds\" (that is, the number of seconds since midnight 1/1/1970, in UTC)."
  }, {
    "name" : "startRecordingPadding",
    "type" : {
      "type" : "int",
      "minInclusive" : 0,
      "hierarchy" : "/seconds/"
    },
    "doc" : "The number of seconds recommended to start the recording early in order to make sure that you catch the begining. Used mainly in Australia and Latin America where the broadcasters often drift or deviate from published show times."
  }, {
    "name" : "duration",
    "type" : {
      "type" : "int",
      "minInclusive" : 0,
      "hierarchy" : "/seconds/"
    },
    "doc" : "The duration of this showing, in seconds."
  }, {
    "name" : "totalRecordingDuration",
    "type" : [ "null", {
      "type" : "int",
      "minInclusive" : 0,
      "hierarchy" : "/seconds/"
    } ],
    "doc" : "If this showing is linked to the next showing, such that a user who records this showing will also want the next showing, then this field will be the total duration of all linked showings (the total time to record in order to capture all of the relevant showings). In practice, only used in Europe and other markets where shows may be broken up by an \"intermission\". If not set, the recording should use the `duration`.",
    "default" : null
  }, {
    "name" : "endRecordingPadding",
    "type" : {
      "type" : "int",
      "minInclusive" : 0,
      "hierarchy" : "/seconds/"
    },
    "doc" : "The number of seconds recommended to continue the recording longer in order to make sure that you catch the end. Used mainly in Australia and Latin America where the broadcasters often drift or deviate from published show times."
  }, {
    "name" : "premiereOrFinale",
    "type" : [ "null", {
      "type" : "enum",
      "name" : "BookendValue",
      
      "doc" : "Types of Premiere and Finale showings which \"bookend\" seasons or series.",
      "symbols" : [ "Premiere", "SeasonPremiere", "SeriesPremiere", "Finale", "SeasonFinale", "SeriesFinale" ],
      "hierarchy" : "/BookendValue/"
    } ],
    "doc" : "A marker if this showing is a Premiere or Finale.",
    "default" : null
  }, {
    "name" : "live",
    "type" : [ "null", {
      "type" : "enum",
      "name" : "LiveValue",
      
      "doc" : "Types of live showings.",
      "symbols" : [ "Delay", "Tape", "Live" ],
      "hierarchy" : "/LiveValue/"
    } ],
    "doc" : "A marker for the type of delay for this program if it is a broadcast of a live event.",
    "default" : null
  }, {
    "name" : "dolby",
    "type" : [ "null", {
      "type" : "enum",
      "name" : "DolbyValue",
      
      "doc" : "The types of collections.",
      "symbols" : [ "Dolby", "DolbyDigital" ],
      "hierarchy" : "/DolbyValue/"
    } ],
    "doc" : "Dolby digital marker, if relevant.",
    "default" : null
  }, {
    "name" : "resolution",
    "type" : {
      "type" : "enum",
      "name" : "ResolutionValue",
      
      "doc" : "Types of video resolution.",
      "symbols" : [ "SD", "HD", "UHD" ],
      "hierarchy" : "/ResolutionValue/"
    },
    "doc" : "The video resolution of the showing. See also the `flags`."
  }, {
    "name" : "internalRating",
    "type" : {
      "type" : "map",
      "values" : {
        "type" : "record",
        "name" : "internalRatingValue",
        "doc" : "The values of an Internal Rating map.",
        "fields" : [ {
          "name" : "ratingValueID",
          "type" : [ "null", {
            "type" : "int",
            "minInclusive" : 0,
            "hierarchy" : "/internalRatingValueID/"
          } ],
          "doc" : "The internal rating value ID. Optional in case some records have advisories without values.",
          "default" : null
        }, {
          "name" : "advisoryIDs",
          "type" : {
            "type" : "array",
            "items" : {
              "type" : "int",
              "minInclusive" : 0,
              "hierarchy" : "/internalRatingAdvisoryID/"
            }
          },
          "doc" : "The internal rating advisory IDs.",
          "default" : [ ]
        } ],
        "hierarchy" : "/internalRatingValue/"
      }
    },
    "doc" : "Various Parental Ratings for this content, normalized to an internal TiVo form. The map keys are Internal Rating Type IDs (strings of integers). The map values are the internal rating values.",
    "default" : { }
  }, {
    "name" : "flags",
    "type" : {
      "type" : "array",
      "items" : {
        "type" : "enum",
        "name" : "ShowingFlagsValue",
        
        "doc" : "These represent the bit flags used on a Showing to signal specific characteristics of the showing. They exist here in this combined enumeration for historical reasons.",
        "symbols" : [ "CloseCaption", "Stereo", "Subtitle", "JoinedInProgress", "Sap", "Blackout", "ThreeD", "Repeat", "Letterbox", "Hdtv", "Dvs", "SexRating", "ViolenceRating", "LanguageRating", "DialogRating", "FvRating", "VodProgram", "Hdtv1080P", "PayPerView", "SignLang", "Catchup", "ProgBreak", "AdultRating" ],
        "hierarchy" : "/ShowingFlagsValue/"
      }
    },
    "doc" : "All \"showing flags\" for this showing. These represent bit flags marked as `1` in the showing's bit flag field.",
    "default" : [ ]
  }, {
    "name" : "streamingRules",
    "type" : {
      "type" : "map",
      "values" : {
        "type" : "record",
        "name" : "streamingRule",
        "doc" : "The values of an Streaming Rules map.",
        "fields" : [ {
          "name" : "catchUpAvailable",
          "type" : "boolean",
          "doc" : "Is catchup available from this MSO for this showing?"
        }, {
          "name" : "startOverAvailable",
          "type" : "boolean",
          "doc" : "Is start over available from this MSO for this showing?"
        }, {
          "name" : "restrictionType",
          "type" : {
            "type" : "enum",
            "name" : "StreamingRestrictionTypeValue",
            
            "doc" : "Are the streaming restrictions captured as a whitelist (ALLOW), a blacklist (BLOCKED), or as a live-API call to confirm rights with the MSO (CHECK WITH MSO)?",
            "symbols" : [ "Allowed", "Blocked", "CheckWithMSO" ],
            "hierarchy" : "/StreamingRestrictionTypeValue/"
          },
          "doc" : "Controls whether or not the `capabilities` are whitelisted, blacklisted or delegated."
        }, {
          "name" : "capabilities",
          "type" : [ "null", {
            "type" : "enum",
            "name" : "StreamingCapabilityValue",
            
            "doc" : "Various streaming capabilities.",
            "symbols" : [ "InHomeSideLoading", "InHomeStreaming", "OutOfHomeSideLoading", "OutOfHomeStreaming", "PremiumContentSideLoading" ],
            "hierarchy" : "/StreamingCapabilityValue/"
          } ],
          "doc" : "The various capabilities that are allowed. The meaning of these flags depend on the `restrictionType`: these flags may represent a whitelist of allowed capabilities, a blacklist of blocked capabilities, or a situation where no capabilities are provided here, because each one has to be checked with the MSO.",
          "default" : null
        } ],
        "hierarchy" : "/streamingRule/"
      }
    },
    "doc" : "A map of MSO IDs to streaming rights for this showing. The map keys are MSO IDs (Strings of Integers). The map values are the streaming rules for this showing for an MSO.",
    "default" : { }
  }, {
    "name" : "showingChecksum",
    "type" : [ "null", {
      "type" : "string",
      "avro.java.string" : "String"
    } ],
    "doc" : "An optional checksum for the state of this entity.",
    "default" : null
  } ],
  "hierarchy" : "/root/Showing/"
}
         }
      },
      {
         "name":"contents",
         "type":{
            "type":"array",
            "items":
{
  "type" : "record",
  "name" : "Content",
  "namespace" : "com.tivo.udf.pgd.api.model",
  "doc" : "Content represents a showable program, such as an episode, movie or special. Note that a series master can also manifest as content; it is used to mark showings of the series without knowledge about the specific episode that will be shown.",
  "fields" : [ {
    "name" : "version",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "\d{1,3}\.\d{1,3}(\.\d{1,3})?"
    },
    "doc" : "This is the version of the _writer's PGD API Schema_ at the time the messages was created."
  }, {
    "name" : "generated",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "\d{1,3}\.\d{1,3}(\.\d{1,3})?"
    },
    "doc" : "This is the version of the UDF PGD API Transformer that produced this message."
  }, {
    "name" : "sourceID",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "([a-zA-Z][a-zA-Z0-9_.]*)\$([a-z_][a-z0-9_.]*)(\![a-z_][a-z0-9_.]*)+\)(.+)"
    },
    "doc" : "This is the internal UDF \"Resource ID\" from which this document was generated. This can be very helpful for debugging. The format is `{entity}${primary}`, where `{entity}` is the UDF type (e.g. \"Episode\"\") and `{primary}` is the UDF Primary value (e.g. \"tivo!c!r)123\"). Note: when the primary starts with \"tivo!c!r)\", then the number that follows is the original Metadata Rovi 2.0 Content ID (that is, the sourceProgramId from the Ref Server)."
  }, {
    "name" : "contentID",
    "type" : {
      "type" : "long",
      "minExclusive" : 0,
      "hierarchy" : "/trioObjectID/"
    },
    "doc" : "The content Trio Object ID (a.k.a the program ID)."
  }, {
    "name" : "epgProviderID",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String"
    },
    "doc" : "The EPG Provider ID. For most entities, this will be the \"TMS ID\", which is of the form: 2-letter type prefix, 10-digit collection MFS ID, dash, 10-digit content MFS ID."
  }, {
    "name" : "collectionID",
    "type" : {
      "type" : "long",
      "minExclusive" : 0,
      "hierarchy" : "/trioObjectID/"
    },
    "doc" : "The Trio Object ID for the Collection that this content belongs to (a.k.a the series ID)."
  }, {
    "name" : "title",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String"
    },
    "doc" : "The title for the program. For movies, series, and other's, this will always align with the 'TITLE' column in the Program table."
  }, {
    "name" : "subtitle",
    "type" : [ "null", {
      "type" : "string",
      "avro.java.string" : "String"
    } ],
    "doc" : "The title for an episode. For movies, series, and other's, this will always be null. However, for episodes, this field may be non-null. This this will align with the 'EPISODETITLE' column.",
    "default" : null
  }, {
    "name" : "description",
    "type" : [ "null", {
      "type" : "string",
      "avro.java.string" : "String"
    } ],
    "doc" : "The description for this entity.",
    "default" : null
  }, {
    "name" : "descriptionLanguage",
    "type" : [ "null", {
      "type" : "string",
      "avro.java.string" : "String"
    } ],
    "doc" : "The ISO-639-2 3-letter code and optional ISO-ALPHA-3 country code for the description language. Examples: \"eng\", \"eng-USA\".",
    "default" : null
  }, {
    "name" : "contentType",
    "type" : [ "null", {
      "type" : "enum",
      "name" : "ContentTypeValue",
      
      "doc" : "The types of content.",
      "symbols" : [ "Episode", "Movie", "Series", "Special" ],
      "hierarchy" : "/ContentTypeValue/"
    } ],
    "doc" : "Indicates the program type for this piece of content: episode, series, movie, or special.",
    "default" : null
  }, {
    "name" : "categoryIDs",
    "type" : {
      "type" : "array",
      "items" : {
        "type" : "long",
        "minInclusive" : 0,
        "hierarchy" : "/categoryID/"
      }
    },
    "doc" : "The categories (a.k.a. genres) associated with this content, in priority order.",
    "default" : [ ]
  }, {
    "name" : "releaseYear",
    "type" : [ "null", {
      "type" : "int",
      "minInclusive" : 1000,
      "maxInclusive" : 3000,
      "hierarchy" : "/year/"
    } ],
    "doc" : "The year that this content was originally released.",
    "default" : null
  }, {
    "name" : "duration",
    "type" : [ "null", {
      "type" : "int",
      "minInclusive" : 0,
      "hierarchy" : "/seconds/"
    } ],
    "doc" : "The duration of the content, in seconds.",
    "default" : null
  }, {
    "name" : "originalAirDate",
    "type" : {
      "type" : "map",
      "values" : {
        "type" : "int",
        "logicalType" : "date"
      }
    },
    "doc" : "A map of original air dates, indexed by country. The map keys are ISO-ALPHA-3 country codes, which will represent the country for the original air date. The map values are the original air date (in epoch days) for this content in this country.",
    "default" : { }
  }, {
    "name" : "seasonNumber",
    "type" : [ "null", {
      "type" : "int",
      "minInclusive" : 1
    } ],
    "doc" : "The season number that this episode belongs to.",
    "default" : null
  }, {
    "name" : "episodeNumbers",
    "type" : {
      "type" : "array",
      "items" : {
        "type" : "int",
        "minInclusive" : 1
      }
    },
    "doc" : "The episode number (or numbers) of this content within it's series. Thus is usually a single value. However, it can hold multiple values for situations where the content represents multiple episodes at once. The episode numbers are ordered.",
    "default" : [ ]
  }, {
    "name" : "starRating",
    "type" : {
      "type" : "enum",
      "name" : "StarRatingValue",
      
      "doc" : "An editorial quality rating. NOTE: Capped at 4 stars, not 5. This enum will be fixed in the future.",
      "symbols" : [ "None", "Zero", "PointFive", "One", "OnePointFive", "Two", "TwoPointFive", "Three", "ThreePointFive", "Four", "FourPointFive", "Five" ],
      "hierarchy" : "/StarRatingValue/"
    },
    "doc" : "An editorial rating of quality for this content."
  }, {
    "name" : "internalRating",
    "type" : {
      "type" : "map",
      "values" : {
        "type" : "record",
        "name" : "internalRatingValue",
        "doc" : "The values of an Internal Rating map.",
        "fields" : [ {
          "name" : "ratingValueID",
          "type" : [ "null", {
            "type" : "int",
            "minInclusive" : 0,
            "hierarchy" : "/internalRatingValueID/"
          } ],
          "doc" : "The internal rating value ID. Optional in case some records have advisories without values.",
          "default" : null
        }, {
          "name" : "advisoryIDs",
          "type" : {
            "type" : "array",
            "items" : {
              "type" : "int",
              "minInclusive" : 0,
              "hierarchy" : "/internalRatingAdvisoryID/"
            }
          },
          "doc" : "The internal rating advisory IDs.",
          "default" : [ ]
        } ],
        "hierarchy" : "/internalRatingValue/"
      }
    },
    "doc" : "Various Parental Ratings for this content, normalized to an internal TiVo form. The map keys are Internal Rating Type IDs (strings of integers). The map values are the internal rating values.",
    "default" : { }
  }, {
    "name" : "castList",
    "type" : {
      "type" : "array",
      "items" : {
        "type" : "record",
        "name" : "cast",
        "doc" : "An individual cast or crew entry.",
        "fields" : [ {
          "name" : "personId",
          "type" : {
            "type" : "long",
            "minExclusive" : 0,
            "hierarchy" : "/trioObjectID/"
          },
          "doc" : "The Trio Object ID for the Person."
        }, {
          "name" : "first",
          "type" : [ "null", {
            "type" : "string",
            "avro.java.string" : "String"
          } ],
          "doc" : "The first name of the person (Optional). May also include nick names, middle names, etc.",
          "default" : null
        }, {
          "name" : "last",
          "type" : {
            "type" : "string",
            "avro.java.string" : "String"
          },
          "doc" : "The last name of the person (Required)."
        }, {
          "name" : "roles",
          "type" : {
            "type" : "array",
            "items" : {
              "type" : "enum",
              "name" : "RoleValue",
              
              "doc" : "The types of roles in Cast/Crew.",
              "symbols" : [ "Actor", "Anchor", "Artist", "Choreographer", "Contestant", "Correspondent", "Director", "ExecutiveProducer", "GuestArtist", "GuestStar", "Host", "Judge", "MusicalGuest", "Narrator", "Producer", "Voice", "Writer" ],
              "hierarchy" : "/RoleValue/"
            }
          },
          "doc" : "The responsibilities this person had in relation to creating this content, ordered by importance.",
          "default" : [ ]
        } ],
        "hierarchy" : "/cast/"
      }
    },
    "doc" : "The cast/crew for this content, in priority order.",
    "default" : [ ]
  }, {
    "name" : "contentChecksum",
    "type" : [ "null", {
      "type" : "string",
      "avro.java.string" : "String"
    } ],
    "doc" : "An optional checksum for the state of this entity.",
    "default" : null
  } ],
  "hierarchy" : "/root/Content/"
}
         }
      },
      {
         "name":"collections",
         "type":{
            "type":"array",
            "items":
{
  "type" : "record",
  "name" : "Collection",
  "namespace" : "com.tivo.udf.pgd.api.model",
  "doc" : "Collection represents a container of one or more contents, such as a series, movie, or one-time-only special.",
  "fields" : [ {
    "name" : "version",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "\d{1,3}\.\d{1,3}(\.\d{1,3})?"
    },
    "doc" : "This is the version of the _writer's PGD API Schema_ at the time the messages was created."
  }, {
    "name" : "generated",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "\d{1,3}\.\d{1,3}(\.\d{1,3})?"
    },
    "doc" : "This is the version of the UDF PGD API Transformer that produced this message."
  }, {
    "name" : "sourceID",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String",
      "pattern" : "([a-zA-Z][a-zA-Z0-9_.]*)\$([a-z_][a-z0-9_.]*)(\![a-z_][a-z0-9_.]*)+\)(.+)"
    },
    "doc" : "This is the internal UDF \"Resource ID\" from which this document was generated. This can be very helpful for debugging. The format is `{entity}${primary}`, where `{entity}` is the UDF type (e.g. \"Episode\"\") and `{primary}` is the UDF Primary value (e.g. \"tivo!c!r)123\"). Note: when the primary starts with \"tivo!c!r)\", then the number that follows is the original Metadata Rovi 2.0 Content ID (that is, the sourceProgramId from the Ref Server)."
  }, {
    "name" : "collectionID",
    "type" : {
      "type" : "long",
      "minExclusive" : 0,
      "hierarchy" : "/trioObjectID/"
    },
    "doc" : "The collection Trio Object ID (a.k.a the series ID)."
  }, {
    "name" : "epgProviderID",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String"
    },
    "doc" : "The EPG Provider ID. For most entities, this will be the \"TMS ID\", which is of the form: 2-letter type prefix and a 10-digit collection MFS ID."
  }, {
    "name" : "collectionType",
    "type" : {
      "type" : "enum",
      "name" : "CollectionTypeValue",
      
      "doc" : "The types of collections.",
      "symbols" : [ "Movie", "Series", "Special" ],
      "hierarchy" : "/CollectionTypeValue/"
    },
    "doc" : "The type of collection this is."
  }, {
    "name" : "episodic",
    "type" : "boolean",
    "doc" : "True if this collection is based on a Series Master that contains episodes (that is, if the collectionType is `Series`); False if this collection is a container of one content for other program types, such as Movies (that is, if the collection type is not `Series`)."
  }, {
    "name" : "title",
    "type" : {
      "type" : "string",
      "avro.java.string" : "String"
    },
    "doc" : "The title for this entity."
  }, {
    "name" : "shortTitle",
    "type" : [ "null", {
      "type" : "string",
      "avro.java.string" : "String"
    } ],
    "doc" : "The short title for this collection. A short title is never longer than 20 characters.",
    "default" : null
  }, {
    "name" : "categoryIDs",
    "type" : {
      "type" : "array",
      "items" : {
        "type" : "long",
        "minInclusive" : 0,
        "hierarchy" : "/categoryID/"
      }
    },
    "doc" : "The categories (a.k.a. genres) associated with this collection, in priority order.",
    "default" : [ ]
  }, {
    "name" : "collectionChecksum",
    "type" : [ "null", {
      "type" : "string",
      "avro.java.string" : "String"
    } ],
    "doc" : "An optional checksum for the state of this entity.",
    "default" : null
  } ],
  "hierarchy" : "/root/Collection/"
}
         }
      },
      {
         "name":"showingDeletes",
         "type":{
            "type":"array",
            "items":"long"
         },
         "default": []
      },
      {
         "name":"contentDeletes",
         "type":{
            "type":"array",
            "items":"long"
         },
         "default": []
      },
      {
         "name":"collectionDeletes",
         "type":{
            "type":"array",
            "items":"long"
         },
         "default": []
      }
   ]
}