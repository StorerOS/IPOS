import React from "react"
import { shallow } from "enzyme"
import { ObjectsListContainer } from "../ObjectsListContainer"

describe("ObjectsList", () => {
  it("should render without crashing", () => {
    shallow(<ObjectsListContainer objects={[]} />)
  })

  it("should render ObjectsList with objects", () => {
    const wrapper = shallow(
      <ObjectsListContainer
        objects={[{ name: "test1.jpg" }, { name: "test2.jpg" }]}
      />
    )
    expect(wrapper.find("ObjectsList").length).toBe(1)
    expect(wrapper.find("ObjectsList").prop("objects")).toEqual([
      { name: "test1.jpg" },
      { name: "test2.jpg" }
    ])
  })

  it("should show the loading indicator when the objects are being loaded", () => {
    const wrapper = shallow(
      <ObjectsListContainer
        currentBucket="test1"
        objects={[]}
        listLoading={true}
      />
    )
    expect(wrapper.find(".loading").exists()).toBeTruthy()
  })
})
