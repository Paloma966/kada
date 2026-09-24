import { describe, expect, it } from "vitest";
import type { Span } from "./aiAnswer";
import { parseAnswer } from "./aiAnswer";

const spans = (text: string): Span[] => parseAnswer(text)[0].spans;

describe("parseAnswer", () => {
  it("turns **bold** into a bold span instead of literal asterisks", () => {
    expect(spans("**短链接**是核心")).toEqual([
      { kind: "bold", text: "短链接" },
      { kind: "text", text: "是核心" },
    ]);
  });

  it("keeps an unclosed ** bold while the answer is still streaming", () => {
    expect(spans("正在**生成")).toEqual([
      { kind: "text", text: "正在" },
      { kind: "bold", text: "生成" },
    ]);
  });

  it("keeps a lone marker as text rather than dropping it", () => {
    expect(spans("结尾的 **")).toEqual([{ kind: "text", text: "结尾的 **" }]);
    expect(spans("*强调*")).toEqual([{ kind: "text", text: "*强调*" }]);
    expect(spans("反引号 ` 单独出现")).toEqual([
      { kind: "text", text: "反引号 ` 单独出现" },
    ]);
  });

  it("parses inline code", () => {
    expect(spans("用 `docker logs` 看日志")).toEqual([
      { kind: "text", text: "用 " },
      { kind: "code", text: "docker logs" },
      { kind: "text", text: " 看日志" },
    ]);
  });

  it("classifies headings, bullets and numbered items", () => {
    expect(parseAnswer("### 小结")).toEqual([
      { kind: "heading", marker: "", spans: [{ kind: "text", text: "小结" }] },
    ]);
    expect(parseAnswer("- 生成短链")).toEqual([
      { kind: "bullet", marker: "", spans: [{ kind: "text", text: "生成短链" }] },
    ]);
    expect(parseAnswer("* 查询统计")).toEqual([
      { kind: "bullet", marker: "", spans: [{ kind: "text", text: "查询统计" }] },
    ]);
    expect(parseAnswer("3. 第三步")).toEqual([
      { kind: "ordered", marker: "3.", spans: [{ kind: "text", text: "第三步" }] },
    ]);
  });

  it("keeps blank lines so paragraphs stay separated", () => {
    const lines = parseAnswer("第一段\n\n第二段");
    expect(lines.map((line) => line.spans.length)).toEqual([1, 0, 1]);
  });

  it("leaves an answer with no markers as one text span per line", () => {
    expect(parseAnswer("就一句话")).toEqual([
      { kind: "text", marker: "", spans: [{ kind: "text", text: "就一句话" }] },
    ]);
  });

  it("renders a realistic answer without leaving an asterisk on screen", () => {
    const answer = [
      "目前知识库中**没有可用的资料内容**（检索结果为空）。",
      "",
      "你可以：",
      "- 直接说明你想了解的内容",
      "- 把相关资料补充进知识库",
      "",
      "**需要我帮你查一下账号的短链总览数据吗？**",
    ].join("\n");

    const rendered = parseAnswer(answer).flatMap((line) => line.spans.map((s) => s.text));
    expect(rendered.join("")).not.toContain("*");
    expect(rendered).toContain("没有可用的资料内容");
    expect(rendered).toContain("需要我帮你查一下账号的短链总览数据吗？");
  });
});
