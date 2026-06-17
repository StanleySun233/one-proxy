import {request} from './client';

export type NodeReleaseTags = {
  imageRepo: string;
  latestTag: string;
  tags: string[];
};

export function getNodeReleaseTags() {
  return request<NodeReleaseTags>('/node-release-tags');
}
