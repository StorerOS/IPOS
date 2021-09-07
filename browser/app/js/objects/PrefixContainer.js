import React from "react"
import { connect } from "react-redux"
import ObjectItem from "./ObjectItem"
import * as actionsObjects from "./actions"

export const PrefixContainer = ({ object, currentPrefix, selectPrefix }) => {
  const props = {
    name: object.name,
    contentType: object.contentType,
    onClick: () => selectPrefix(`${currentPrefix}${object.name}`)
  }

  return <ObjectItem {...props} />
}

const mapStateToProps = (state, ownProps) => {
  return {
    object: ownProps.object,
    currentPrefix: state.objects.currentPrefix
  }
}

const mapDispatchToProps = dispatch => {
  return {
    selectPrefix: prefix => dispatch(actionsObjects.selectPrefix(prefix))
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(PrefixContainer)
