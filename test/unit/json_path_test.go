package unit_test

import (
	"bytes"

	"github.com/oliveagle/jsonpath"

	"data-pipelines-worker/types/dataclasses"
)

func (suite *UnitTestSuite) TestJSONPathTranscriptionSegments() {
	tts_transcription := `{
		"duration": 21.690000534057617,
		"language": "english",
		"segments": [
			{
				"avg_logprob": -0.29121363162994385,
				"compression_ratio": 1.4727272987365723,
				"end": 8.140000343322754,
				"id": 0,
				"no_speech_prob": 0.00016069135745055974,
				"seek": 0,
				"start": 0,
				"temperature": 0,
				"text": " On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.",
				"tokens": [
					50364,
					1282,
					7617,
					1025,
					11,
					39498,
					11,
					264,
					1002,
					390,
					5680,
					3105,
					382,
					264,
					38376,
					4736,
					641,
					13828,
					2167,
					294,
					264,
					7051,
					13,
					50771
				],
				"transient": false
			},
			{
				"avg_logprob": -0.29121363162994385,
				"compression_ratio": 1.4727272987365723,
				"end": 12.899999618530273,
				"id": 1,
				"no_speech_prob": 0.00016069135745055974,
				"seek": 0,
				"start": 8.140000343322754,
				"temperature": 0,
				"text": " This marked the start of their legendary musical journey, leading to global fame.",
				"tokens": [
					50771,
					639,
					12658,
					264,
					722,
					295,
					641,
					16698,
					9165,
					4671,
					11,
					5775,
					281,
					4338,
					16874,
					13,
					51009
				],
				"transient": false
			},
			{
				"avg_logprob": -0.29121363162994385,
				"compression_ratio": 1.4727272987365723,
				"end": 16.739999771118164,
				"id": 2,
				"no_speech_prob": 0.00016069135745055974,
				"seek": 0,
				"start": 12.899999618530273,
				"temperature": 0,
				"text": " Interestingly, John Lennon's harmonica playing added a distinct touch,",
				"tokens": [
					51009,
					30564,
					11,
					2619,
					441,
					1857,
					266,
					311,
					14750,
					2262,
					2433,
					3869,
					257,
					10644,
					2557,
					11,
					51201
				],
				"transient": false
			},
			{
				"avg_logprob": -0.29121363162994385,
				"compression_ratio": 1.4727272987365723,
				"end": 20.540000915527344,
				"id": 3,
				"no_speech_prob": 0.00016069135745055974,
				"seek": 0,
				"start": 16.739999771118164,
				"temperature": 0,
				"text": " propelling them toward unprecedented stardom in the music industry.",
				"tokens": [
					51201,
					25577,
					2669,
					552,
					7361,
					21555,
					342,
					515,
					298,
					294,
					264,
					1318,
					3518,
					13,
					51391
				],
				"transient": false
			}
		],
		"task": "transcribe",
		"text": "On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK. This marked the start of their legendary musical journey, leading to global fame. Interestingly, John Lennon's harmonica playing added a distinct touch, propelling them toward unprecedented stardom in the music industry.",
		"words": null
	}`

	jsonData, err := dataclasses.HandleResultValue(
		bytes.NewBufferString(tts_transcription).Bytes(),
	)
	suite.Nil(err)
	suite.NotNil(jsonData)

	jsonPathData, err := jsonpath.JsonPathLookup(
		jsonData,
		"$.segments[*].text",
	)
	suite.Nil(err)
	suite.NotEmpty(jsonPathData)
	suite.Equal([]interface{}{
		" On October 5, 1962, the world was forever changed as the Beatles released their debut single in the UK.",
		" This marked the start of their legendary musical journey, leading to global fame.",
		" Interestingly, John Lennon's harmonica playing added a distinct touch,",
		" propelling them toward unprecedented stardom in the music industry.",
	}, jsonPathData)

}
