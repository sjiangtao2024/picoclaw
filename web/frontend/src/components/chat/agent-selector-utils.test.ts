import assert from "node:assert/strict"
import test from "node:test"

import {
  buildDefaultAgentPatch,
  getCurrentDefaultAgentId,
  listAvailableAgents,
} from "./agent-selector-utils.ts"

test("listAvailableAgents returns named agents from config", () => {
  const agents = listAvailableAgents({
    agents: {
      list: [
        { id: "dayahuan", name: "大丫鬟", default: true },
        { id: "image-agent", name: "生图助理", default: false },
      ],
    },
  })

  assert.deepEqual(agents, [
    { id: "dayahuan", name: "大丫鬟", isDefault: true },
    { id: "image-agent", name: "生图助理", isDefault: false },
  ])
})

test("getCurrentDefaultAgentId falls back to the first agent when none is marked default", () => {
  assert.equal(
    getCurrentDefaultAgentId({
      agents: {
        list: [
          { id: "dayahuan", name: "大丫鬟" },
          { id: "image-agent", name: "生图助理" },
        ],
      },
    }),
    "dayahuan",
  )
})

test("buildDefaultAgentPatch marks only the selected agent as default", () => {
  const patch = buildDefaultAgentPatch(
    {
      agents: {
        list: [
          { id: "dayahuan", name: "大丫鬟", default: true, workspace: "/a" },
          { id: "image-agent", name: "生图助理", default: false, workspace: "/b" },
        ],
      },
    },
    "image-agent",
  )

  assert.deepEqual(patch, {
    agents: {
      list: [
        {
          id: "dayahuan",
          name: "大丫鬟",
          default: false,
          workspace: "/a",
        },
        {
          id: "image-agent",
          name: "生图助理",
          default: true,
          workspace: "/b",
        },
      ],
    },
  })
})

test("buildDefaultAgentPatch rejects unknown target agents", () => {
  assert.throws(
    () =>
      buildDefaultAgentPatch(
        {
          agents: {
            list: [{ id: "dayahuan", name: "大丫鬟", default: true }],
          },
        },
        "missing",
      ),
    /Unknown agent/,
  )
})
