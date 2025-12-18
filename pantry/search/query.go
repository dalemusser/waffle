package search

import (
	"encoding/json"
	"fmt"
)

// QueryBuilder builds search queries.
type QueryBuilder struct {
	must    []any
	should  []any
	mustNot []any
	filter  []any

	minimumShouldMatch any
	boost              *float64
}

// Query creates a new query builder.
func Query() *QueryBuilder {
	return &QueryBuilder{}
}

// Must adds a must clause (AND).
func (q *QueryBuilder) Must(clauses ...any) *QueryBuilder {
	q.must = append(q.must, clauses...)
	return q
}

// Should adds a should clause (OR).
func (q *QueryBuilder) Should(clauses ...any) *QueryBuilder {
	q.should = append(q.should, clauses...)
	return q
}

// MustNot adds a must_not clause (NOT).
func (q *QueryBuilder) MustNot(clauses ...any) *QueryBuilder {
	q.mustNot = append(q.mustNot, clauses...)
	return q
}

// Filter adds a filter clause (no scoring).
func (q *QueryBuilder) Filter(clauses ...any) *QueryBuilder {
	q.filter = append(q.filter, clauses...)
	return q
}

// MinimumShouldMatch sets the minimum number of should clauses that must match.
func (q *QueryBuilder) MinimumShouldMatch(value any) *QueryBuilder {
	q.minimumShouldMatch = value
	return q
}

// Boost sets the boost value for the query.
func (q *QueryBuilder) Boost(value float64) *QueryBuilder {
	q.boost = &value
	return q
}

// Build builds the query as a map.
func (q *QueryBuilder) Build() map[string]any {
	// If only one clause with no other conditions, return it directly
	if len(q.must) == 1 && len(q.should) == 0 && len(q.mustNot) == 0 && len(q.filter) == 0 {
		return map[string]any{"query": q.must[0]}
	}

	// If no clauses at all, return match_all
	if len(q.must) == 0 && len(q.should) == 0 && len(q.mustNot) == 0 && len(q.filter) == 0 {
		return map[string]any{"query": map[string]any{"match_all": map[string]any{}}}
	}

	// Build bool query
	boolQuery := make(map[string]any)

	if len(q.must) > 0 {
		boolQuery["must"] = q.must
	}
	if len(q.should) > 0 {
		boolQuery["should"] = q.should
	}
	if len(q.mustNot) > 0 {
		boolQuery["must_not"] = q.mustNot
	}
	if len(q.filter) > 0 {
		boolQuery["filter"] = q.filter
	}
	if q.minimumShouldMatch != nil {
		boolQuery["minimum_should_match"] = q.minimumShouldMatch
	}
	if q.boost != nil {
		boolQuery["boost"] = *q.boost
	}

	return map[string]any{
		"query": map[string]any{
			"bool": boolQuery,
		},
	}
}

// BuildQuery returns just the query portion (without the wrapper).
func (q *QueryBuilder) BuildQuery() map[string]any {
	built := q.Build()
	if query, ok := built["query"]; ok {
		return query.(map[string]any)
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (q *QueryBuilder) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.Build())
}

// IsEmpty returns true if no clauses have been added.
func (q *QueryBuilder) IsEmpty() bool {
	return len(q.must) == 0 && len(q.should) == 0 && len(q.mustNot) == 0 && len(q.filter) == 0
}

// --- Query clauses ---

// MatchAll returns a match_all query.
func MatchAll() map[string]any {
	return map[string]any{"match_all": map[string]any{}}
}

// MatchNone returns a match_none query.
func MatchNone() map[string]any {
	return map[string]any{"match_none": map[string]any{}}
}

// Match returns a match query.
func Match(field string, query any) map[string]any {
	return map[string]any{
		"match": map[string]any{
			field: query,
		},
	}
}

// MatchWithOptions returns a match query with options.
func MatchWithOptions(field string, query any, opts map[string]any) map[string]any {
	fieldOpts := map[string]any{"query": query}
	for k, v := range opts {
		fieldOpts[k] = v
	}
	return map[string]any{
		"match": map[string]any{
			field: fieldOpts,
		},
	}
}

// MatchPhrase returns a match_phrase query.
func MatchPhrase(field string, query any) map[string]any {
	return map[string]any{
		"match_phrase": map[string]any{
			field: query,
		},
	}
}

// MatchPhrasePrefix returns a match_phrase_prefix query.
func MatchPhrasePrefix(field string, query any) map[string]any {
	return map[string]any{
		"match_phrase_prefix": map[string]any{
			field: query,
		},
	}
}

// MultiMatch returns a multi_match query.
func MultiMatch(query string, fields ...string) *MultiMatchQuery {
	return &MultiMatchQuery{
		query:  query,
		fields: fields,
	}
}

// MultiMatchQuery builds a multi_match query.
type MultiMatchQuery struct {
	query              string
	fields             []string
	matchType          string
	operator           string
	minimumShouldMatch any
	fuzziness          any
	boost              *float64
	analyzer           string
	tieBreaker         *float64
}

// Type sets the multi_match type.
func (m *MultiMatchQuery) Type(t string) *MultiMatchQuery {
	m.matchType = t
	return m
}

// Operator sets the operator (and/or).
func (m *MultiMatchQuery) Operator(op string) *MultiMatchQuery {
	m.operator = op
	return m
}

// Fuzziness sets the fuzziness.
func (m *MultiMatchQuery) Fuzziness(f any) *MultiMatchQuery {
	m.fuzziness = f
	return m
}

// Boost sets the boost value.
func (m *MultiMatchQuery) Boost(b float64) *MultiMatchQuery {
	m.boost = &b
	return m
}

// Analyzer sets the analyzer.
func (m *MultiMatchQuery) Analyzer(a string) *MultiMatchQuery {
	m.analyzer = a
	return m
}

// TieBreaker sets the tie breaker.
func (m *MultiMatchQuery) TieBreaker(t float64) *MultiMatchQuery {
	m.tieBreaker = &t
	return m
}

// MinimumShouldMatch sets minimum_should_match.
func (m *MultiMatchQuery) MinimumShouldMatch(v any) *MultiMatchQuery {
	m.minimumShouldMatch = v
	return m
}

// Build builds the multi_match query.
func (m *MultiMatchQuery) Build() map[string]any {
	query := map[string]any{
		"query":  m.query,
		"fields": m.fields,
	}

	if m.matchType != "" {
		query["type"] = m.matchType
	}
	if m.operator != "" {
		query["operator"] = m.operator
	}
	if m.fuzziness != nil {
		query["fuzziness"] = m.fuzziness
	}
	if m.boost != nil {
		query["boost"] = *m.boost
	}
	if m.analyzer != "" {
		query["analyzer"] = m.analyzer
	}
	if m.tieBreaker != nil {
		query["tie_breaker"] = *m.tieBreaker
	}
	if m.minimumShouldMatch != nil {
		query["minimum_should_match"] = m.minimumShouldMatch
	}

	return map[string]any{"multi_match": query}
}

// MarshalJSON implements json.Marshaler.
func (m *MultiMatchQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Build())
}

// Term returns a term query (exact match).
func Term(field string, value any) map[string]any {
	return map[string]any{
		"term": map[string]any{
			field: value,
		},
	}
}

// Terms returns a terms query (multiple exact matches, OR).
func Terms(field string, values ...any) map[string]any {
	return map[string]any{
		"terms": map[string]any{
			field: values,
		},
	}
}

// Exists returns an exists query.
func Exists(field string) map[string]any {
	return map[string]any{
		"exists": map[string]any{
			"field": field,
		},
	}
}

// Prefix returns a prefix query.
func Prefix(field, value string) map[string]any {
	return map[string]any{
		"prefix": map[string]any{
			field: value,
		},
	}
}

// Wildcard returns a wildcard query.
func Wildcard(field, value string) map[string]any {
	return map[string]any{
		"wildcard": map[string]any{
			field: value,
		},
	}
}

// Regexp returns a regexp query.
func Regexp(field, value string) map[string]any {
	return map[string]any{
		"regexp": map[string]any{
			field: value,
		},
	}
}

// Fuzzy returns a fuzzy query.
func Fuzzy(field, value string) *FuzzyQuery {
	return &FuzzyQuery{
		field: field,
		value: value,
	}
}

// FuzzyQuery builds a fuzzy query.
type FuzzyQuery struct {
	field            string
	value            string
	fuzziness        any
	prefixLength     *int
	maxExpansions    *int
	transpositions   *bool
	rewrite          string
	boost            *float64
}

// Fuzziness sets the fuzziness.
func (f *FuzzyQuery) Fuzziness(v any) *FuzzyQuery {
	f.fuzziness = v
	return f
}

// PrefixLength sets the prefix length.
func (f *FuzzyQuery) PrefixLength(v int) *FuzzyQuery {
	f.prefixLength = &v
	return f
}

// MaxExpansions sets max expansions.
func (f *FuzzyQuery) MaxExpansions(v int) *FuzzyQuery {
	f.maxExpansions = &v
	return f
}

// Transpositions enables/disables transpositions.
func (f *FuzzyQuery) Transpositions(v bool) *FuzzyQuery {
	f.transpositions = &v
	return f
}

// Boost sets the boost value.
func (f *FuzzyQuery) Boost(v float64) *FuzzyQuery {
	f.boost = &v
	return f
}

// Build builds the fuzzy query.
func (f *FuzzyQuery) Build() map[string]any {
	opts := map[string]any{"value": f.value}

	if f.fuzziness != nil {
		opts["fuzziness"] = f.fuzziness
	}
	if f.prefixLength != nil {
		opts["prefix_length"] = *f.prefixLength
	}
	if f.maxExpansions != nil {
		opts["max_expansions"] = *f.maxExpansions
	}
	if f.transpositions != nil {
		opts["transpositions"] = *f.transpositions
	}
	if f.boost != nil {
		opts["boost"] = *f.boost
	}

	return map[string]any{
		"fuzzy": map[string]any{
			f.field: opts,
		},
	}
}

// MarshalJSON implements json.Marshaler.
func (f *FuzzyQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Build())
}

// Range returns a range query builder.
func Range(field string) *RangeQuery {
	return &RangeQuery{field: field}
}

// RangeQuery builds a range query.
type RangeQuery struct {
	field    string
	gte      any
	gt       any
	lte      any
	lt       any
	format   string
	timezone string
	boost    *float64
}

// Gte sets greater than or equal.
func (r *RangeQuery) Gte(v any) *RangeQuery {
	r.gte = v
	return r
}

// Gt sets greater than.
func (r *RangeQuery) Gt(v any) *RangeQuery {
	r.gt = v
	return r
}

// Lte sets less than or equal.
func (r *RangeQuery) Lte(v any) *RangeQuery {
	r.lte = v
	return r
}

// Lt sets less than.
func (r *RangeQuery) Lt(v any) *RangeQuery {
	r.lt = v
	return r
}

// Between sets both gte and lte.
func (r *RangeQuery) Between(from, to any) *RangeQuery {
	r.gte = from
	r.lte = to
	return r
}

// Format sets the date format.
func (r *RangeQuery) Format(f string) *RangeQuery {
	r.format = f
	return r
}

// Timezone sets the timezone.
func (r *RangeQuery) Timezone(tz string) *RangeQuery {
	r.timezone = tz
	return r
}

// Boost sets the boost value.
func (r *RangeQuery) Boost(v float64) *RangeQuery {
	r.boost = &v
	return r
}

// Build builds the range query.
func (r *RangeQuery) Build() map[string]any {
	opts := make(map[string]any)

	if r.gte != nil {
		opts["gte"] = r.gte
	}
	if r.gt != nil {
		opts["gt"] = r.gt
	}
	if r.lte != nil {
		opts["lte"] = r.lte
	}
	if r.lt != nil {
		opts["lt"] = r.lt
	}
	if r.format != "" {
		opts["format"] = r.format
	}
	if r.timezone != "" {
		opts["time_zone"] = r.timezone
	}
	if r.boost != nil {
		opts["boost"] = *r.boost
	}

	return map[string]any{
		"range": map[string]any{
			r.field: opts,
		},
	}
}

// MarshalJSON implements json.Marshaler.
func (r *RangeQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Build())
}

// Bool returns a nested bool query.
func Bool() *QueryBuilder {
	return Query()
}

// IDs returns an ids query.
func IDs(ids ...string) map[string]any {
	return map[string]any{
		"ids": map[string]any{
			"values": ids,
		},
	}
}

// QueryString returns a query_string query.
func QueryString(query string) *QueryStringQuery {
	return &QueryStringQuery{query: query}
}

// QueryStringQuery builds a query_string query.
type QueryStringQuery struct {
	query              string
	defaultField       string
	fields             []string
	defaultOperator    string
	analyzer           string
	fuzziness          any
	minimumShouldMatch any
	boost              *float64
}

// DefaultField sets the default field.
func (q *QueryStringQuery) DefaultField(f string) *QueryStringQuery {
	q.defaultField = f
	return q
}

// Fields sets the fields to search.
func (q *QueryStringQuery) Fields(fields ...string) *QueryStringQuery {
	q.fields = fields
	return q
}

// DefaultOperator sets the default operator.
func (q *QueryStringQuery) DefaultOperator(op string) *QueryStringQuery {
	q.defaultOperator = op
	return q
}

// Analyzer sets the analyzer.
func (q *QueryStringQuery) Analyzer(a string) *QueryStringQuery {
	q.analyzer = a
	return q
}

// Fuzziness sets the fuzziness.
func (q *QueryStringQuery) Fuzziness(f any) *QueryStringQuery {
	q.fuzziness = f
	return q
}

// MinimumShouldMatch sets minimum_should_match.
func (q *QueryStringQuery) MinimumShouldMatch(v any) *QueryStringQuery {
	q.minimumShouldMatch = v
	return q
}

// Boost sets the boost value.
func (q *QueryStringQuery) Boost(b float64) *QueryStringQuery {
	q.boost = &b
	return q
}

// Build builds the query_string query.
func (q *QueryStringQuery) Build() map[string]any {
	opts := map[string]any{"query": q.query}

	if q.defaultField != "" {
		opts["default_field"] = q.defaultField
	}
	if len(q.fields) > 0 {
		opts["fields"] = q.fields
	}
	if q.defaultOperator != "" {
		opts["default_operator"] = q.defaultOperator
	}
	if q.analyzer != "" {
		opts["analyzer"] = q.analyzer
	}
	if q.fuzziness != nil {
		opts["fuzziness"] = q.fuzziness
	}
	if q.minimumShouldMatch != nil {
		opts["minimum_should_match"] = q.minimumShouldMatch
	}
	if q.boost != nil {
		opts["boost"] = *q.boost
	}

	return map[string]any{"query_string": opts}
}

// MarshalJSON implements json.Marshaler.
func (q *QueryStringQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.Build())
}

// SimpleQueryString returns a simple_query_string query.
func SimpleQueryString(query string) *SimpleQueryStringQuery {
	return &SimpleQueryStringQuery{query: query}
}

// SimpleQueryStringQuery builds a simple_query_string query.
type SimpleQueryStringQuery struct {
	query           string
	fields          []string
	defaultOperator string
	analyzer        string
	flags           string
	boost           *float64
}

// Fields sets the fields to search.
func (q *SimpleQueryStringQuery) Fields(fields ...string) *SimpleQueryStringQuery {
	q.fields = fields
	return q
}

// DefaultOperator sets the default operator.
func (q *SimpleQueryStringQuery) DefaultOperator(op string) *SimpleQueryStringQuery {
	q.defaultOperator = op
	return q
}

// Analyzer sets the analyzer.
func (q *SimpleQueryStringQuery) Analyzer(a string) *SimpleQueryStringQuery {
	q.analyzer = a
	return q
}

// Flags sets the flags.
func (q *SimpleQueryStringQuery) Flags(f string) *SimpleQueryStringQuery {
	q.flags = f
	return q
}

// Boost sets the boost value.
func (q *SimpleQueryStringQuery) Boost(b float64) *SimpleQueryStringQuery {
	q.boost = &b
	return q
}

// Build builds the simple_query_string query.
func (q *SimpleQueryStringQuery) Build() map[string]any {
	opts := map[string]any{"query": q.query}

	if len(q.fields) > 0 {
		opts["fields"] = q.fields
	}
	if q.defaultOperator != "" {
		opts["default_operator"] = q.defaultOperator
	}
	if q.analyzer != "" {
		opts["analyzer"] = q.analyzer
	}
	if q.flags != "" {
		opts["flags"] = q.flags
	}
	if q.boost != nil {
		opts["boost"] = *q.boost
	}

	return map[string]any{"simple_query_string": opts}
}

// MarshalJSON implements json.Marshaler.
func (q *SimpleQueryStringQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.Build())
}

// Nested returns a nested query.
func Nested(path string, query any) map[string]any {
	return map[string]any{
		"nested": map[string]any{
			"path":  path,
			"query": query,
		},
	}
}

// NestedWithScoreMode returns a nested query with score mode.
func NestedWithScoreMode(path string, query any, scoreMode string) map[string]any {
	return map[string]any{
		"nested": map[string]any{
			"path":       path,
			"query":      query,
			"score_mode": scoreMode,
		},
	}
}

// HasChild returns a has_child query.
func HasChild(childType string, query any) map[string]any {
	return map[string]any{
		"has_child": map[string]any{
			"type":  childType,
			"query": query,
		},
	}
}

// HasParent returns a has_parent query.
func HasParent(parentType string, query any) map[string]any {
	return map[string]any{
		"has_parent": map[string]any{
			"parent_type": parentType,
			"query":       query,
		},
	}
}

// GeoDistance returns a geo_distance query.
func GeoDistance(field string, lat, lon float64, distance string) map[string]any {
	return map[string]any{
		"geo_distance": map[string]any{
			"distance": distance,
			field: map[string]any{
				"lat": lat,
				"lon": lon,
			},
		},
	}
}

// GeoBoundingBox returns a geo_bounding_box query.
func GeoBoundingBox(field string, topLat, topLon, bottomLat, bottomLon float64) map[string]any {
	return map[string]any{
		"geo_bounding_box": map[string]any{
			field: map[string]any{
				"top_left": map[string]any{
					"lat": topLat,
					"lon": topLon,
				},
				"bottom_right": map[string]any{
					"lat": bottomLat,
					"lon": bottomLon,
				},
			},
		},
	}
}

// MoreLikeThis returns a more_like_this query.
func MoreLikeThis(fields []string, likeText string) map[string]any {
	return map[string]any{
		"more_like_this": map[string]any{
			"fields": fields,
			"like":   likeText,
		},
	}
}

// Script returns a script query.
func Script(source string) map[string]any {
	return map[string]any{
		"script": map[string]any{
			"script": map[string]any{
				"source": source,
			},
		},
	}
}

// ScriptWithParams returns a script query with parameters.
func ScriptWithParams(source string, params map[string]any) map[string]any {
	return map[string]any{
		"script": map[string]any{
			"script": map[string]any{
				"source": source,
				"params": params,
			},
		},
	}
}

// Boosting returns a boosting query.
func Boosting(positive, negative any, negativeBoost float64) map[string]any {
	return map[string]any{
		"boosting": map[string]any{
			"positive":       positive,
			"negative":       negative,
			"negative_boost": negativeBoost,
		},
	}
}

// ConstantScore wraps a filter in a constant_score query.
func ConstantScore(filter any, boost float64) map[string]any {
	return map[string]any{
		"constant_score": map[string]any{
			"filter": filter,
			"boost":  boost,
		},
	}
}

// FunctionScore creates a function_score query builder.
func FunctionScore(query any) *FunctionScoreQuery {
	return &FunctionScoreQuery{query: query}
}

// FunctionScoreQuery builds a function_score query.
type FunctionScoreQuery struct {
	query       any
	functions   []map[string]any
	scoreMode   string
	boostMode   string
	maxBoost    *float64
	minScore    *float64
}

// AddFunction adds a scoring function.
func (f *FunctionScoreQuery) AddFunction(fn map[string]any) *FunctionScoreQuery {
	f.functions = append(f.functions, fn)
	return f
}

// ScoreMode sets the score mode.
func (f *FunctionScoreQuery) ScoreMode(mode string) *FunctionScoreQuery {
	f.scoreMode = mode
	return f
}

// BoostMode sets the boost mode.
func (f *FunctionScoreQuery) BoostMode(mode string) *FunctionScoreQuery {
	f.boostMode = mode
	return f
}

// MaxBoost sets the maximum boost.
func (f *FunctionScoreQuery) MaxBoost(v float64) *FunctionScoreQuery {
	f.maxBoost = &v
	return f
}

// MinScore sets the minimum score threshold.
func (f *FunctionScoreQuery) MinScore(v float64) *FunctionScoreQuery {
	f.minScore = &v
	return f
}

// Build builds the function_score query.
func (f *FunctionScoreQuery) Build() map[string]any {
	result := map[string]any{"query": f.query}

	if len(f.functions) > 0 {
		result["functions"] = f.functions
	}
	if f.scoreMode != "" {
		result["score_mode"] = f.scoreMode
	}
	if f.boostMode != "" {
		result["boost_mode"] = f.boostMode
	}
	if f.maxBoost != nil {
		result["max_boost"] = *f.maxBoost
	}
	if f.minScore != nil {
		result["min_score"] = *f.minScore
	}

	return map[string]any{"function_score": result}
}

// MarshalJSON implements json.Marshaler.
func (f *FunctionScoreQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Build())
}

// --- Aggregations ---

// Agg creates a new aggregation builder.
func Agg() *AggregationBuilder {
	return &AggregationBuilder{
		aggs: make(map[string]any),
	}
}

// AggregationBuilder builds aggregations.
type AggregationBuilder struct {
	aggs map[string]any
}

// Terms adds a terms aggregation.
func (a *AggregationBuilder) Terms(name, field string, size int) *AggregationBuilder {
	agg := map[string]any{
		"field": field,
	}
	if size > 0 {
		agg["size"] = size
	}
	a.aggs[name] = map[string]any{"terms": agg}
	return a
}

// Histogram adds a histogram aggregation.
func (a *AggregationBuilder) Histogram(name, field string, interval float64) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"histogram": map[string]any{
			"field":    field,
			"interval": interval,
		},
	}
	return a
}

// DateHistogram adds a date_histogram aggregation.
func (a *AggregationBuilder) DateHistogram(name, field, interval string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"date_histogram": map[string]any{
			"field":             field,
			"calendar_interval": interval,
		},
	}
	return a
}

// Range adds a range aggregation.
func (a *AggregationBuilder) Range(name, field string, ranges []map[string]any) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"range": map[string]any{
			"field":  field,
			"ranges": ranges,
		},
	}
	return a
}

// Avg adds an avg aggregation.
func (a *AggregationBuilder) Avg(name, field string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"avg": map[string]any{"field": field},
	}
	return a
}

// Sum adds a sum aggregation.
func (a *AggregationBuilder) Sum(name, field string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"sum": map[string]any{"field": field},
	}
	return a
}

// Min adds a min aggregation.
func (a *AggregationBuilder) Min(name, field string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"min": map[string]any{"field": field},
	}
	return a
}

// Max adds a max aggregation.
func (a *AggregationBuilder) Max(name, field string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"max": map[string]any{"field": field},
	}
	return a
}

// Cardinality adds a cardinality aggregation.
func (a *AggregationBuilder) Cardinality(name, field string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"cardinality": map[string]any{"field": field},
	}
	return a
}

// Stats adds a stats aggregation.
func (a *AggregationBuilder) Stats(name, field string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"stats": map[string]any{"field": field},
	}
	return a
}

// ExtendedStats adds an extended_stats aggregation.
func (a *AggregationBuilder) ExtendedStats(name, field string) *AggregationBuilder {
	a.aggs[name] = map[string]any{
		"extended_stats": map[string]any{"field": field},
	}
	return a
}

// TopHits adds a top_hits aggregation.
func (a *AggregationBuilder) TopHits(name string, size int, sort []SortOption) *AggregationBuilder {
	topHits := map[string]any{"size": size}
	if len(sort) > 0 {
		sortList := make([]map[string]any, len(sort))
		for i, s := range sort {
			sortList[i] = map[string]any{s.Field: map[string]any{"order": s.Order}}
		}
		topHits["sort"] = sortList
	}
	a.aggs[name] = map[string]any{"top_hits": topHits}
	return a
}

// Nested adds a nested aggregation.
func (a *AggregationBuilder) Nested(name, path string, subAggs map[string]any) *AggregationBuilder {
	nested := map[string]any{
		"nested": map[string]any{"path": path},
	}
	if subAggs != nil {
		nested["aggs"] = subAggs
	}
	a.aggs[name] = nested
	return a
}

// Filter adds a filter aggregation.
func (a *AggregationBuilder) Filter(name string, filter any, subAggs map[string]any) *AggregationBuilder {
	filterAgg := map[string]any{
		"filter": filter,
	}
	if subAggs != nil {
		filterAgg["aggs"] = subAggs
	}
	a.aggs[name] = filterAgg
	return a
}

// Custom adds a custom aggregation.
func (a *AggregationBuilder) Custom(name string, agg map[string]any) *AggregationBuilder {
	a.aggs[name] = agg
	return a
}

// Build returns the aggregations map.
func (a *AggregationBuilder) Build() map[string]any {
	return a.aggs
}

// --- Common aggregation result types ---

// TermsBucket represents a bucket in a terms aggregation.
type TermsBucket struct {
	Key      any   `json:"key"`
	DocCount int64 `json:"doc_count"`
}

// TermsAggResult represents the result of a terms aggregation.
type TermsAggResult struct {
	Buckets []TermsBucket `json:"buckets"`
}

// StatsAggResult represents the result of a stats aggregation.
type StatsAggResult struct {
	Count int64   `json:"count"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Sum   float64 `json:"sum"`
}

// ValueAggResult represents a single value aggregation result.
type ValueAggResult struct {
	Value float64 `json:"value"`
}

// Stringify helpers for debugging

// String returns a JSON representation of the query.
func (q *QueryBuilder) String() string {
	data, _ := json.MarshalIndent(q.Build(), "", "  ")
	return string(data)
}

// FormatQuery formats a query map as JSON string.
func FormatQuery(query map[string]any) string {
	data, err := json.MarshalIndent(query, "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(data)
}
