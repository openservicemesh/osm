#include <string>
#include <unordered_map>

#include "proxy_wasm_intrinsics.h"

class ExampleContext : public Context
{
public:
  explicit ExampleContext(uint32_t id, RootContext *root) : Context(id, root) {}

  FilterHeadersStatus onRequestHeaders(uint32_t headers, bool end_of_stream) override;
};
static RegisterContextFactory register_ExampleContext(CONTEXT_FACTORY(ExampleContext));

FilterHeadersStatus ExampleContext::onRequestHeaders(uint32_t headers, bool end_of_stream)
{
  LOG_INFO("JON WAS HERE");
  return FilterHeadersStatus::Continue;
}
