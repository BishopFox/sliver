import lunr from "lunr";
import React from "react";
import { Doc, Docs } from "./docs";


export class SearchCtx {

  private _docs: Docs = { docs: [] };
  private _docsIndex: lunr.Index;

  constructor() {
    this._docsIndex = lunr(function () {
      this.ref("name");
      this.field("content");
    });
  }

  public searchDocs = (query: string): Doc[] => {
    const results = this._docsIndex.search(query);
    const docs = results.map((result) => {
      return this._docs.docs.find((doc) => doc.name === result.ref);
    });
    return docs.filter((doc) => doc !== undefined) as Doc[];
  }

  public addDocs = (docs: Docs) => {
    this._docs = docs;
    this._docsIndex = lunr(function () {
      this.ref("name");
      this.field("content");
      docs.docs.forEach((doc) => {
        this.add(doc);
      });
    });
  }

}

export const SearchContext = React.createContext<SearchCtx>(new SearchCtx());
export const useSearchContext = () => React.useContext(SearchContext);
