#include <string>
#include <unordered_map>

#include "proxy_wasm_intrinsics.h"

static bool isInbound()
{
  int64_t direction = 0;
  getValue({"listener_direction"}, &direction);
  return direction == 1;
}

using RqTotalCounter = Counter<std::string, std::string, std::string, std::string, std::string, std::string, std::string, std::string, std::string>;
using RqDurationHist = Histogram<std::string, std::string, std::string, std::string, std::string, std::string, std::string, std::string>;

class StatsContext : public Context
{
public:
  explicit StatsContext(uint32_t id, RootContext *root) : Context(id, root),
                                                          rq_total(RqTotalCounter::New("osm_request_total",
                                                                                       "response_code",
                                                                                       "source_namespace",
                                                                                       "source_kind",
                                                                                       "source_name",
                                                                                       "source_pod",
                                                                                       "destination_namespace",
                                                                                       "destination_kind",
                                                                                       "destination_name",
                                                                                       "destination_pod")),
                                                          rq_duration(RqDurationHist::New("osm_request_duration_ms",
                                                                                          "source_namespace",
                                                                                          "source_kind",
                                                                                          "source_name",
                                                                                          "source_pod",
                                                                                          "destination_namespace",
                                                                                          "destination_kind",
                                                                                          "destination_name",
                                                                                          "destination_pod"))
  {
  }

  void onCreate() override;
  void onDone() override;

  FilterHeadersStatus onRequestHeaders(uint32_t headers, bool end_of_stream) override;
  FilterHeadersStatus onResponseHeaders(uint32_t headers, bool end_of_stream) override;

private:
  RqTotalCounter *rq_total;
  RqDurationHist *rq_duration;
  std::string source_pod, source_namespace, source_kind, source_name;
  std::string destination_namespace, destination_kind, destination_name, destination_pod;
  uint64_t start_time;
};
static RegisterContextFactory register_StatsContext(CONTEXT_FACTORY(StatsContext));

void StatsContext::onCreate()
{
  if (isInbound())
  {
    return;
  }

  start_time = getCurrentTimeNanoseconds();
}

void StatsContext::onDone()
{
  if (isInbound())
  {
    return;
  }

  uint64_t duration_ns = getCurrentTimeNanoseconds() - start_time;
  int64_t duration_ms = duration_ns / 1000 / 1000;

  rq_duration->record(duration_ms,
                      source_namespace, source_kind, source_name, source_pod,
                      destination_namespace, destination_kind, destination_name, destination_pod);
}

FilterHeadersStatus StatsContext::onRequestHeaders(uint32_t headers, bool end_of_stream)
{
  if (isInbound())
  {
    return FilterHeadersStatus::Continue;
  }

  source_namespace = getRequestHeader("osm-stats-namespace").get()->toString();
  source_kind = getRequestHeader("osm-stats-kind").get()->toString();
  source_name = getRequestHeader("osm-stats-name").get()->toString();
  source_pod = getRequestHeader("osm-stats-pod").get()->toString();

  removeRequestHeader("osm-stats-namespace");
  removeRequestHeader("osm-stats-kind");
  removeRequestHeader("osm-stats-name");
  removeRequestHeader("osm-stats-pod");

  return FilterHeadersStatus::Continue;
}

FilterHeadersStatus StatsContext::onResponseHeaders(uint32_t headers, bool end_of_stream)
{
  if (isInbound())
  {
    return FilterHeadersStatus::Continue;
  }

  std::string response_code = getResponseHeader(":status").get()->toString();
  destination_namespace = getResponseHeader("osm-stats-namespace").get()->toString();
  destination_kind = getResponseHeader("osm-stats-kind").get()->toString();
  destination_name = getResponseHeader("osm-stats-name").get()->toString();
  destination_pod = getResponseHeader("osm-stats-pod").get()->toString();

  rq_total->increment(1, response_code,
                      source_namespace, source_kind, source_name, source_pod,
                      destination_namespace, destination_kind, destination_name, destination_pod);

  removeResponseHeader("osm-stats-namespace");
  removeResponseHeader("osm-stats-kind");
  removeResponseHeader("osm-stats-name");
  removeResponseHeader("osm-stats-pod");

  return FilterHeadersStatus::Continue;
}
