import MarkdownViewer from "@/components/markdown";
import { useSearchContext } from "@/util/search-context";
import { faChevronCircleRight } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, Card, CardBody, Divider } from "@heroui/react";
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
    <div className="px-4 pb-8 pt-4">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-4">
        <div className="text-3xl">
          Search: &quot;{query?.slice(0, 50)}&quot;
          <div className="text-sm text-gray-500">
            {searchResults.length} Results
          </div>
        </div>

        {searchResults.map((doc) => (
          <Card key={doc.name}>
            <CardBody className="flex flex-col gap-3">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
                <span className="text-xl">{doc.name}</span>
                <div className="flex-1"></div>
                <Button
                  size="sm"
                  color="secondary"
                  className="w-full sm:w-[150px]"
                  onPress={() => {
                    router.push(`/docs?name=${doc.name}`);
                  }}
                >
                  Full Doc
                  <FontAwesomeIcon icon={faChevronCircleRight} />
                </Button>
              </div>

              <Divider />

              <div className="max-h-[200px] overflow-hidden">
                <MarkdownViewer markdown={doc.content} />
              </div>
            </CardBody>
          </Card>
        ))}
      </div>
    </div>
  );
};

export default SearchPage;
