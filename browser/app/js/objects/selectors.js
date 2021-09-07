import { createSelector } from "reselect"

export const getCurrentPrefix = state => state.objects.currentPrefix

export const getCheckedList = state => state.objects.checkedList

export const getPrefixWritable = state => state.objects.prefixWritable
