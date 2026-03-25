import GenericToolInterface from "@/components/studio/GenericToolInterface";

export default function GenericToolPage({ params }: { params: { id: string } }) {
  return <GenericToolInterface params={params} />;
}
