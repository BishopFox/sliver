import MarkdownViewer from "@/components/markdown";
import { useSearchContext } from "@/util/search-context";
import { faChevronCircleRight } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, Card, CardBody, Divider } from "@nextui-org/react";
import { NextPage } from "next";
import { useSearchParams } from "next/navigation";
import { useRouter } from "next/router";
import React from "react";

export type SearchPageProps = {};

const SearchPage: NextPage = (props: SearchPageProps) => {
  const router = useRouter();
  const search = useSearchContext();
  const query = useSearchParams().get("search");

  const searchResults = React.useMemo(() => {
    if (query) {
      return search.searchDocs(query);
    }
    return [];
  }, [query, search]);

  return (
    <div className="grid grid-cols-12 mt-2">
      <div className="col-span-1"></div>
      <div className="col-span-10">
        <div className="text-3xl">
          Search: &quot;{query?.slice(0, 50)}&quot;
          <div className="text-sm text-gray-500">
            {searchResults.length} Results
          </div>
        </div>

        {searchResults.map((doc) => (
          <Card key={doc.name} className="mt-4">
            <CardBody>
              <div className="flex">
                <span className="text-xl">{doc.name}</span>
                <div className="flex-grow"></div>
                <Button
                  size="sm"
                  color="secondary"
                  className="w-[150px]"
                  onPress={() => {
                    router.push(`/docs?name=${doc.name}`);
                  }}
                >
                  Full Doc
                  <FontAwesomeIcon icon={faChevronCircleRight} />
                </Button>
              </div>

              <Divider className="mt-1" />

              <div className="mt-2 overflow-hidden max-h-[200px]">
                <MarkdownViewer markdown={doc.content} />
              </div>
            </CardBody>
          </Card>
        ))}
      </div>
      <div className="col-span-1"></div>

      <div className="col-span-12 mb-8"></div>
    </div>
  );
};

export default SearchPage;
